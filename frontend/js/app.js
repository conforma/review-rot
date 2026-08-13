const state = {
    data: [],
    leaderboard: null,
    leaderboardDays: null,
    generatedAt: null,
    view: 'prs',
    filters: {
        type: 'regular',
        author: 'all',
        repo: 'all',
        readyForReview: false
    },
    sort: {
        field: 'updated_at',
        direction: 'desc'
    }
};

async function init() {
    try {
        restoreFiltersFromURL();

        const response = await fetch('data.json');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const json = await response.json();
        state.data = json.pull_requests || [];
        state.leaderboard = json.leaderboard || null;
        state.generatedAt = json.generated_at;
        applyUISettings(json.ui_settings);
        populateFilters();
        initSortArrows();
        render();
        buildIntervalSelector();
        renderLeaderboard();
        reflectView();
        updateFooter();
        attachEventListeners();
    } catch (err) {
        console.error('Failed to load data:', err);
        document.getElementById('error-banner').hidden = false;
        document.querySelector('.table-container').hidden = true;
    }
}

function applyUISettings(settings) {
    if (!settings) return;
    const root = document.documentElement.style;

    if (settings.title) {
        document.getElementById('header-title').textContent = settings.title;
        document.getElementById('page-title').textContent = settings.title;
    }

    if (settings.logo) {
        const logo = document.getElementById('header-logo');
        logo.src = settings.logo;
        logo.alt = settings.title || 'Logo';
        logo.hidden = false;
    }

    if (settings.favicon) {
        document.getElementById('favicon').href = settings.favicon;
    }

    if (settings.palette) {
        if (settings.palette.accent) root.setProperty('--accent', settings.palette.accent);
        if (settings.palette.accent_dark) root.setProperty('--accent-dark', settings.palette.accent_dark);
        if (settings.palette.accent_light) root.setProperty('--accent-light', settings.palette.accent_light);
    }
}

function populateFilters() {
    const authors = [...new Set(state.data.map(pr => (pr.author || {}).login).filter(Boolean))].sort();
    const repos = [...new Set(state.data.map(pr => pr.repo).filter(Boolean))].sort();

    const authorSelect = document.getElementById('author-filter');
    authors.forEach(author => {
        const opt = document.createElement('option');
        opt.value = author;
        opt.textContent = author;
        authorSelect.appendChild(opt);
    });

    const repoSelect = document.getElementById('repo-filter');
    repos.forEach(repo => {
        const opt = document.createElement('option');
        opt.value = repo;
        opt.textContent = repo;
        repoSelect.appendChild(opt);
    });

    if (state.filters.author !== 'all') authorSelect.value = state.filters.author;
    if (state.filters.repo !== 'all') repoSelect.value = state.filters.repo;
}

function restoreFiltersFromURL() {
    const params = new URLSearchParams(window.location.search);

    const view = params.get('view');
    if (view === 'leaderboard') state.view = 'leaderboard';

    const days = parseInt(params.get('days'), 10);
    if (Number.isFinite(days) && days >= 0) state.leaderboardDays = days;

    if (params.get('ready') === '1') {
        state.filters.readyForReview = true;
        document.getElementById('ready-filter').checked = true;
    }

    const type = params.get('type');
    if (type && ['regular', 'automated', 'wip', 'all'].includes(type)) {
        state.filters.type = type;
        document.querySelectorAll('.type-btn').forEach(b => {
            b.classList.toggle('active', b.dataset.type === type);
        });
    }

    const author = params.get('author');
    if (author) state.filters.author = author;

    const repo = params.get('repo');
    if (repo) state.filters.repo = repo;

    const sortField = params.get('sort');
    const sortDir = params.get('dir');
    if (sortField) {
        state.sort.field = sortField;
        state.sort.direction = sortDir === 'desc' ? 'desc' : 'asc';
    }
}

function updateURL() {
    const url = new URL(window.location);
    const p = url.searchParams;

    const setOrDelete = (key, value, defaultValue) => {
        if (value !== defaultValue) p.set(key, value);
        else p.delete(key);
    };

    setOrDelete('view', state.view, 'prs');

    // The horizon (full window) is the default, so it stays out of the URL.
    const horizon = (state.leaderboard && state.leaderboard.window_days) || null;
    if (state.leaderboardDays != null && state.leaderboardDays !== horizon) {
        p.set('days', state.leaderboardDays);
    } else {
        p.delete('days');
    }

    setOrDelete('type', state.filters.type, 'regular');
    setOrDelete('author', state.filters.author, 'all');
    setOrDelete('repo', state.filters.repo, 'all');
    state.filters.readyForReview ? p.set('ready', '1') : p.delete('ready');

    if (state.sort.field) {
        p.set('sort', state.sort.field);
        setOrDelete('dir', state.sort.direction, 'asc');
    } else {
        p.delete('sort');
        p.delete('dir');
    }

    history.replaceState(null, '', url);
}

function attachEventListeners() {
    document.querySelectorAll('.view-tab').forEach(btn => {
        btn.addEventListener('click', () => {
            setView(btn.dataset.view);
        });
    });

    const lbBody = document.getElementById('leaderboard-body');
    lbBody.addEventListener('click', (e) => {
        if (e.target.closest('a')) return; // let profile / PR links work
        const row = e.target.closest('.lb-row');
        if (row) toggleLeaderboardRow(row);
    });
    lbBody.addEventListener('keydown', (e) => {
        if (e.key !== 'Enter' && e.key !== ' ') return;
        const row = e.target.closest('.lb-row');
        if (row && e.target === row) {
            e.preventDefault();
            toggleLeaderboardRow(row);
        }
    });

    document.querySelectorAll('.type-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.type-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            state.filters.type = btn.dataset.type;
            render();
            updateURL();
        });
    });

    document.getElementById('author-filter').addEventListener('change', e => {
        state.filters.author = e.target.value;
        render();
        updateURL();
    });

    document.getElementById('repo-filter').addEventListener('change', e => {
        state.filters.repo = e.target.value;
        render();
        updateURL();
    });

    document.getElementById('ready-filter').addEventListener('change', e => {
        state.filters.readyForReview = e.target.checked;
        render();
        updateURL();
    });

    document.querySelectorAll('th.sortable').forEach(th => {
        th.tabIndex = 0;
        th.setAttribute('role', 'button');
        th.addEventListener('click', () => {
            const field = th.dataset.sort;
            if (state.sort.field === field) {
                if (state.sort.direction === 'asc') {
                    state.sort.direction = 'desc';
                } else {
                    state.sort.field = null;
                    state.sort.direction = 'asc';
                }
            } else {
                state.sort.field = field;
                state.sort.direction = 'asc';
            }
            render();
            updateURL();
        });
        th.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                th.click();
            }
        });
    });
}

function filterPRs(prs, filters) {
    return prs.filter(pr => {
        const isWip = pr.is_draft || /\bWIP\b/i.test(pr.title);
        if (filters.type === 'regular' && (pr.is_automated || isWip)) return false;
        if (filters.type === 'automated' && !pr.is_automated) return false;
        if (filters.type === 'wip' && !isWip) return false;

        if (filters.author !== 'all' && pr.author.login !== filters.author) return false;
        if (filters.repo !== 'all' && pr.repo !== filters.repo) return false;

        if (filters.readyForReview) {
            if (pr.is_draft) return false;
            if (pr.ci_status !== 'SUCCESS') return false;
            const hasUnreviewedChanges = pr.reviews.count === 0 || pr.reviews.has_new_commits;
            if (!hasUnreviewedChanges) return false;
        }

        return true;
    });
}

function sortPRs(prs, sort) {
    if (!sort.field) return [...prs];
    return [...prs].sort((a, b) => {
        let cmp = 0;
        const ciOrder = { SUCCESS: 0, PENDING: 1, ERROR: 1, FAILURE: 2 };
        switch (sort.field) {
            case 'created_at': cmp = new Date(a.created_at) - new Date(b.created_at); break;
            case 'updated_at': cmp = new Date(a.updated_at) - new Date(b.updated_at); break;
            case 'reviews': cmp = a.reviews.count - b.reviews.count; break;
            case 'title': cmp = a.title.localeCompare(b.title); break;
            case 'ci_status': cmp = (ciOrder[a.ci_status] ?? 3) - (ciOrder[b.ci_status] ?? 3); break;
            case 'threads': cmp = a.unresolved_conversations - b.unresolved_conversations; break;
            case 're_review': {
                const needsA = (a.reviews.count === 0 || a.reviews.has_new_commits) ? 1 : 0;
                const needsB = (b.reviews.count === 0 || b.reviews.has_new_commits) ? 1 : 0;
                cmp = needsA - needsB;
                break;
            }
        }
        return sort.direction === 'asc' ? cmp : -cmp;
    });
}

function computeStats(filteredPRs) {
    const count = filteredPRs.length;
    const now = Date.now();
    const totalAge = filteredPRs.reduce((sum, pr) => sum + (now - new Date(pr.created_at)), 0);
    const avgAge = count > 0 ? totalAge / count : 0;
    return { count, avgAge };
}

function formatElapsed(isoDate) {
    const hours = Math.floor((Date.now() - new Date(isoDate)) / 3600000);
    if (hours < 1) return '<1h';
    if (hours < 24) return `${hours}h`;
    return `${Math.floor(hours / 24)}d`;
}

function formatDuration(ms) {
    const hours = Math.floor(ms / 3600000);
    if (hours < 24) return `${hours}h`;
    return `${Math.floor(hours / 24)}d`;
}

function escapeHtml(str) {
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function ageColorClass(isoDate) {
    const days = (Date.now() - new Date(isoDate)) / 86400000;
    if (days >= 30) return 'age-critical';
    if (days >= 7) return 'age-old';
    if (days >= 3) return 'age-warn';
    return '';
}

function avatarUrl(url) {
    if (!/^https?:\/\//i.test(url)) return '';
    const sep = url.includes('?') ? '&' : '?';
    return `${url}${sep}s=40`;
}

function safeClass(str) {
    return str.replace(/[^a-z0-9-]/g, '');
}

function renderCIStatus(status) {
    const cls = 'ci-' + safeClass((status || 'unknown').toLowerCase());
    return `<span class="ci-dot ${cls}"></span>`;
}

function renderSize(size) {
    if (!size) return '';
    const cls = 'size-' + safeClass(size.toLowerCase());
    return `<span class="size-badge ${cls}">${escapeHtml(size)}</span>`;
}

function renderReReview(reviews) {
    if (reviews.count === 0 || reviews.has_new_commits) {
        return '<span class="re-review-yes">&#x1F440;</span>';
    }
    return '';
}

function renderRow(pr) {
    const author = pr.author || {};
    const reviews = pr.reviews || { count: 0, has_new_commits: false };
    const draftBadge = pr.is_draft ? '<span class="draft-badge">Draft</span>' : '';
    const ageCls = ageColorClass(pr.created_at);
    const ageClass = ageCls ? ` ${ageCls}` : '';
    return `<tr>
        <td class="pr-cell">
            <div class="pr-title-row">
                <a href="${escapeHtml(pr.url || '')}" target="_blank" rel="noopener">${escapeHtml(pr.title || '')}</a>
                ${renderSize(pr.size)}${draftBadge}
            </div>
            <div class="pr-info-line">
                <span class="pr-repo">${escapeHtml(pr.repo || '')}</span>
                <img class="avatar-sm" src="${escapeHtml(avatarUrl(author.avatar_url || ''))}" alt="${escapeHtml(author.login || '')}" width="16" height="16">
                ${escapeHtml(author.login || '')}
            </div>
        </td>
        <td>${renderCIStatus(pr.ci_status)}</td>
        <td class="age-cell">${pr.updated_at ? formatElapsed(pr.updated_at) : ''}</td>
        <td class="age-cell${ageClass}">${pr.created_at ? formatElapsed(pr.created_at) : ''}</td>
        <td>${pr.unresolved_conversations || 0}</td>
        <td class="reviews-cell">${reviews.count}</td>
        <td>${renderReReview(reviews)}</td>
    </tr>`;
}

function initSortArrows() {
    document.querySelectorAll('th.sortable').forEach(th => {
        const arrow = document.createElement('span');
        arrow.className = 'sort-arrow';
        arrow.textContent = '▲ ';
        th.prepend(arrow);
    });
}

function updateSortIndicators() {
    document.querySelectorAll('th.sortable').forEach(th => {
        const arrow = th.querySelector('.sort-arrow');
        if (state.sort.field && th.dataset.sort === state.sort.field) {
            arrow.textContent = state.sort.direction === 'asc' ? '▲' : '▼';
            arrow.style.visibility = 'visible';
        } else {
            arrow.style.visibility = 'hidden';
        }
    });
}

function render() {
    const filtered = filterPRs(state.data, state.filters);
    const sorted = sortPRs(filtered, state.sort);

    const tbody = document.getElementById('pr-table-body');
    const emptyState = document.getElementById('empty-state');
    const tableContainer = document.querySelector('.table-container');

    if (sorted.length === 0) {
        tbody.innerHTML = '';
        tableContainer.hidden = true;
        emptyState.hidden = false;
    } else {
        tbody.innerHTML = sorted.map(renderRow).join('');
        tableContainer.hidden = false;
        emptyState.hidden = true;
    }

    const stats = computeStats(filtered);
    document.getElementById('total-count').textContent = stats.count;
    document.getElementById('avg-age').textContent = stats.count > 0 ? formatDuration(stats.avgAge) : '—';

    updateSortIndicators();
}

function setView(view) {
    state.view = view;
    reflectView();
    updateURL();
}

function reflectView() {
    const isLeaderboard = state.view === 'leaderboard';

    document.querySelectorAll('.view-tab').forEach(b => {
        b.classList.toggle('active', b.dataset.view === state.view);
    });

    document.getElementById('prs-view').hidden = isLeaderboard;
    document.getElementById('leaderboard-view').hidden = !isLeaderboard;

    // The header stats (PRs shown / avg age) only apply to the PR list.
    const stats = document.querySelector('.stats');
    if (stats) stats.style.visibility = isLeaderboard ? 'hidden' : 'visible';
}

// leaderboardForInterval re-counts and re-ranks reviewers for the given window
// by keeping only the PRs each person engaged with since (generated_at - days).
function leaderboardForInterval(days) {
    const lb = state.leaderboard;
    const reviewers = (lb && Array.isArray(lb.reviewers)) ? lb.reviewers : [];
    const ref = state.generatedAt ? new Date(state.generatedAt).getTime() : Date.now();
    const cutoff = ref - days * 86400000;

    return reviewers
        .map(r => {
            const prs = (r.prs || []).filter(pr => {
                const t = new Date(pr.engaged_at).getTime();
                return !Number.isNaN(t) && t >= cutoff;
            });
            return { login: r.login || '', reviews: prs.length, prs };
        })
        .filter(r => r.reviews > 0)
        .sort((a, b) => b.reviews - a.reviews || a.login.localeCompare(b.login));
}

// buildIntervalSelector wires up the day-window slider (0 .. horizon) plus the
// preset shortcuts that snap it to common windows. The horizon is the backend's
// window_days; the whole control is hidden when there is no usable horizon.
function buildIntervalSelector() {
    const slider = document.getElementById('lb-interval-slider');
    const presetsEl = document.getElementById('lb-interval-presets');
    if (!slider || !presetsEl) return;

    const horizon = (state.leaderboard && state.leaderboard.window_days) || 0;
    const group = slider.closest('.filter-group');

    if (horizon <= 0) {
        if (group) group.hidden = true;
        if (state.leaderboardDays == null) state.leaderboardDays = 0;
        return;
    }
    if (group) group.hidden = false;

    if (state.leaderboardDays == null) state.leaderboardDays = horizon;
    state.leaderboardDays = clampDays(state.leaderboardDays, horizon);

    slider.min = 0;
    slider.max = horizon;
    slider.value = state.leaderboardDays;

    // Presets that fit inside the horizon; the horizon itself is always offered.
    const presets = [...new Set([7, 30, 90].filter(d => d < horizon)), horizon].sort((a, b) => a - b);
    presetsEl.innerHTML = presets.map(d =>
        `<button class="type-btn" data-days="${d}">${d}d</button>`
    ).join('');
    presetsEl.querySelectorAll('.type-btn').forEach(btn => {
        btn.addEventListener('click', () => setLeaderboardInterval(Number(btn.dataset.days)));
    });

    slider.addEventListener('input', () => setLeaderboardInterval(Number(slider.value)));

    updateIntervalUI();
}

function clampDays(days, horizon) {
    return Math.max(0, Math.min(horizon, days));
}

function setLeaderboardInterval(days) {
    const horizon = (state.leaderboard && state.leaderboard.window_days) || days;
    state.leaderboardDays = clampDays(days, horizon);
    updateIntervalUI();
    renderLeaderboard();
    updateURL();
}

// updateIntervalUI syncs the slider position, the "N days" readout, the filled
// portion of the track, and which preset (if any) is currently active.
function updateIntervalUI() {
    const slider = document.getElementById('lb-interval-slider');
    const valueEl = document.getElementById('lb-interval-value');
    const days = state.leaderboardDays || 0;

    if (slider && Number(slider.value) !== days) slider.value = days;
    if (valueEl) valueEl.textContent = `${days} ${days === 1 ? 'day' : 'days'}`;

    document.querySelectorAll('#lb-interval-presets .type-btn').forEach(b => {
        b.classList.toggle('active', Number(b.dataset.days) === days);
    });

    if (slider) {
        const max = Number(slider.max) || 1;
        slider.style.setProperty('--lb-slider-pct', `${max > 0 ? (days / max) * 100 : 0}%`);
    }
}

function toggleLeaderboardRow(row) {
    const index = row.dataset.index;
    const detail = document.getElementById(`lb-detail-${index}`);
    if (!detail) return;
    const expanded = row.getAttribute('aria-expanded') === 'true';
    row.setAttribute('aria-expanded', String(!expanded));
    row.classList.toggle('expanded', !expanded);
    detail.hidden = expanded;
}

function renderLeaderboard() {
    const subtitle = document.getElementById('leaderboard-subtitle');
    const body = document.getElementById('leaderboard-body');
    const empty = document.getElementById('leaderboard-empty');
    const container = document.querySelector('#leaderboard-view .table-container');

    const days = state.leaderboardDays || 0;
    const reviewers = leaderboardForInterval(days);

    subtitle.textContent = days > 0
        ? `Distinct PRs each person reviewed or commented on across monitored repos in the last ${days} days`
        : '';

    if (reviewers.length === 0) {
        body.innerHTML = '';
        container.hidden = true;
        empty.hidden = false;
        return;
    }

    container.hidden = false;
    empty.hidden = true;

    const max = reviewers[0].reviews || 1;
    body.innerHTML = reviewers.map((r, i) => renderLeaderboardRow(r, i, max)).join('');
}

function renderLeaderboardRow(reviewer, index, max) {
    const login = reviewer.login || '';
    const reviews = reviewer.reviews || 0;
    const rank = index + 1;
    const rankCell = rank === 1 ? '\u{1F947}' : rank === 2 ? '\u{1F948}' : rank === 3 ? '\u{1F949}' : String(rank);
    const pct = max > 0 ? Math.round((reviews / max) * 100) : 0;
    const profile = `https://github.com/${encodeURIComponent(login)}`;
    const avatar = `${profile}.png?size=40`;

    const rankClass = rank <= 3 ? ` rank-${rank}` : '';
    const mainRow = `<tr class="lb-row" data-index="${index}" tabindex="0" role="button" aria-expanded="false" aria-controls="lb-detail-${index}">
        <td class="lb-rank">${rankCell}</td>
        <td class="lb-reviewer">
            <div class="lb-reviewer-inner">
                <span class="lb-chevron" aria-hidden="true">&#x25B6;</span>
                <img class="avatar-sm" src="${escapeHtml(avatar)}" alt="" width="24" height="24">
                <a href="${escapeHtml(profile)}" target="_blank" rel="noopener">${escapeHtml(login)}</a>
            </div>
        </td>
        <td class="lb-count">
            <div class="lb-count-inner">
                <span class="lb-meter"><span class="lb-meter-fill${rankClass}" style="--lb-w: ${pct}%"></span></span>
                <span class="lb-count-value">${reviews}</span>
            </div>
        </td>
    </tr>`;

    const prs = [...(reviewer.prs || [])].sort((a, b) => (b.engaged_at || '').localeCompare(a.engaged_at || ''));
    const items = prs.map(renderReviewedPR).join('');
    const detailRow = `<tr class="lb-detail" id="lb-detail-${index}" hidden>
        <td class="lb-detail-cell" colspan="3">
            <table class="lb-pr-table">
                <thead>
                    <tr><th>PR</th><th>Repo</th><th>Author</th><th>Reviewed</th></tr>
                </thead>
                <tbody>${items}</tbody>
            </table>
        </td>
    </tr>`;

    return mainRow + detailRow;
}

function renderReviewedPR(pr) {
    const repo = pr.repo || '';
    const number = pr.number || 0;
    const ref = number ? `${repo}#${number}` : repo;
    const author = pr.author || '';
    const authorCell = author
        ? `<a href="https://github.com/${encodeURIComponent(author)}" target="_blank" rel="noopener">${escapeHtml(author)}</a>`
        : '—';
    return `<tr>
        <td class="lb-pr-title"><a href="${escapeHtml(pr.url || '')}" target="_blank" rel="noopener">${escapeHtml(pr.title || ref)}</a></td>
        <td class="lb-pr-repo">${escapeHtml(repo)}</td>
        <td class="lb-pr-author">${authorCell}</td>
        <td class="lb-pr-date">${pr.engaged_at ? escapeHtml(formatDate(pr.engaged_at)) : ''}</td>
    </tr>`;
}

function formatDate(iso) {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return '';
    return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

function formatRelativeTime(date) {
    const seconds = Math.floor((Date.now() - date) / 1000);
    if (seconds < 60) return 'just now';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `~${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `~${hours}h ago`;
    const days = Math.floor(hours / 24);
    return `~${days}d ago`;
}

function updateFooter() {
    if (!state.generatedAt) return;
    const el = document.getElementById('last-updated');
    const date = new Date(state.generatedAt);
    const timeStr = date.toLocaleString(undefined, {
        year: 'numeric', month: 'numeric', day: 'numeric',
        hour: 'numeric', minute: '2-digit', timeZoneName: 'short'
    });
    el.textContent = `Last updated: ${timeStr} (${formatRelativeTime(date)})`;
}

document.addEventListener('DOMContentLoaded', init);
