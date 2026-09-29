const test = require('node:test');
const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { join } = require('node:path');
const vm = require('node:vm');

const app = readFileSync(join(__dirname, 'app.js'), 'utf8');

function loadApp(search = '') {
    const checkbox = { checked: false };
    const window = { location: new URL(`https://example.test/${search}`) };
    const context = vm.createContext({
        window, URL, URLSearchParams,
        document: {
            addEventListener() {},
            getElementById() { return checkbox; },
            querySelectorAll() { return []; }
        },
        history: { replaceState(_state, _title, url) { window.location = new URL(url); } }
    });
    vm.runInContext(app, context, { filename: 'app.js' });
    return { context, state: vm.runInContext('state', context), checkbox, window };
}

function makePR({ reviews = {}, ...fields } = {}) {
    return {
        title: 'PR', url: 'https://github.com/conforma/policy/pull/1',
        repo: 'conforma/policy', author: { login: 'alice', avatar_url: '' },
        created_at: '2025-03-15T10:00:00Z', updated_at: '2025-03-16T10:00:00Z',
        is_draft: false, is_automated: false, ci_status: 'SUCCESS',
        unresolved_conversations: 0,
        reviews: {
            approved_count: 0, has_new_commits: true,
            outstanding_change_requests_on_head: false, ...reviews
        },
        ...fields
    };
}

test('Ready uses draft, CI, current-head requests and approval threshold only', () => {
    const { context, state } = loadApp();
    state.requiredApprovals = 2;
    const filters = { type: 'all', author: 'all', repo: 'all', readyForReview: true };
    for (const [name, fields, wantReady] of [
        ['no review yet', {}, true],
        ['one approval', { reviews: { approved_count: 1, has_new_commits: false } }, true],
        ['approval threshold reached', { reviews: { approved_count: 2 } }, false],
        ['draft', { is_draft: true }, false],
        ['failing CI', { ci_status: 'FAILURE' }, false],
        ['pending CI', { ci_status: 'PENDING' }, false],
        ['missing CI', { ci_status: null }, false],
        ['outstanding current-head request', { reviews: { outstanding_change_requests_on_head: true } }, false],
        ['old-head request only', { unresolved_conversations: 1 }, true]
    ]) {
        assert.equal(context.filterPRs([makePR(fields)], filters).length === 1, wantReady, name);
    }

    const reviewed = makePR({ reviews: { has_new_commits: false } });
    const unreviewed = makePR({ reviews: { has_new_commits: true } });
    assert.equal(context.filterPRs([reviewed], filters).length, 1);
    assert.equal(context.filterPRs([unreviewed], filters).length, 1);
    state.requiredApprovals = 1;
    assert.equal(context.filterPRs([makePR({ reviews: { approved_count: 1 } })], filters).length, 0);
});

test('Ready keeps type, author and repo filters', () => {
    const { context, state } = loadApp();
    state.requiredApprovals = 2;
    const filters = { type: 'regular', author: 'alice', repo: 'conforma/policy', readyForReview: true };
    assert.equal(context.filterPRs([makePR()], filters).length, 1);
    assert.equal(context.filterPRs([makePR({ is_automated: true })], filters).length, 0);
    assert.equal(context.filterPRs([makePR({ title: 'WIP: not done' })], filters).length, 0);
    assert.equal(context.filterPRs([makePR({ author: { login: 'bob' } })], filters).length, 0);
    assert.equal(context.filterPRs([makePR({ repo: 'other/repo' })], filters).length, 0);
    assert.equal(context.filterPRs([makePR({ title: 'WIP: not done' })], { ...filters, type: 'all' }).length, 1);
});

test('table renders and sorts approvals, unresolved total and head coverage separately', () => {
    const { context } = loadApp();
    const oneApproval = makePR({
        title: 'one', unresolved_conversations: 3,
        reviews: { count: 99, approved_count: 1, has_new_commits: false }
    });
    const twoApprovals = makePR({
        title: 'two', reviews: { count: 0, approved_count: 2, has_new_commits: true }
    });
    const prs = [twoApprovals, oneApproval];
    assert.deepEqual(Array.from(context.sortPRs(prs, { field: 'reviews', direction: 'asc' }), pr => pr.title), ['one', 'two']);
    assert.deepEqual(Array.from(context.sortPRs(prs, { field: 'reviews', direction: 'desc' }), pr => pr.title), ['two', 'one']);
    assert.deepEqual(Array.from(context.sortPRs(prs, { field: 're_review', direction: 'asc' }), pr => pr.title), ['one', 'two']);

    const oneRow = context.renderRow(oneApproval);
    assert.match(oneRow, /<td>3<\/td>\s*<td class="reviews-cell">1<\/td>/);
    assert.doesNotMatch(oneRow, /class="re-review-yes"/);
    const twoRow = context.renderRow(twoApprovals);
    assert.match(twoRow, /<td class="reviews-cell">2<\/td>/);
    assert.match(twoRow, /class="re-review-yes"[^>]*aria-label="Unreviewed changes"/);
    assert.match(twoRow, /title="No eligible approval or change request was submitted on the current head"/);
});

test('ready=1 restores the checkbox and sort key, and survives URL updates', () => {
    const { context, state, checkbox, window } = loadApp('?ready=1&sort=reviews');
    context.restoreFiltersFromURL();
    assert.equal(state.filters.readyForReview, true);
    assert.equal(checkbox.checked, true);
    assert.equal(state.sort.field, 'reviews');
    assert.equal(context.filterPRs([makePR({ ci_status: 'FAILURE' })], state.filters).length, 0);
    context.updateURL();
    assert.equal(window.location.searchParams.get('ready'), '1');
    assert.equal(window.location.searchParams.get('sort'), 'reviews');

    state.filters.readyForReview = false;
    checkbox.checked = false;
    assert.equal(context.filterPRs([makePR({ ci_status: 'FAILURE' })], state.filters).length, 1);
    context.updateURL();
    assert.equal(window.location.searchParams.has('ready'), false);
    assert.equal(window.location.searchParams.get('sort'), 'reviews');
});

test('column and filter tooltips describe current-head approvals and combined unresolved', () => {
    const html = readFileSync(join(__dirname, '..', 'index.html'), 'utf8');
    assert.ok(html.includes('data-sort="reviews"><span class="col-tooltip-wrap">Approvals'));
    assert.ok(html.includes('data-sort="threads">Unresolved <span class="col-tooltip-wrap">threads + requests'));
    assert.ok(html.includes('Unresolved inline review threads plus outstanding change requests, including requests on older commits'));
    assert.ok(html.includes('No eligible reviewer has submitted an approval or change request on the current head'));
    assert.ok(html.includes('no outstanding change requests on the current head, and fewer than the required current-head approvals'));
});
