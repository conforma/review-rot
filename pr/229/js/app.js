const state = {
    data: [],
    generatedAt: null,
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
        const params = new URLSearchParams(window.location.search);
        if (params.get('ready') === '1') {
            state.filters.readyForReview = true;
            document.getElementById('ready-filter').checked = true;
        }

        const response = await fetch('data.json');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const json = await response.json();
        state.data = json.pull_requests || [];
        state.generatedAt = json.generated_at;
        populateFilters();
        render();
        updateFooter();
        attachEventListeners();
    } catch (err) {
        document.getElementById('error-banner').hidden = false;
        document.querySelector('.table-container').hidden = true;
    }
}

function populateFilters() {
    const authors = [...new Set(state.data.map(pr => pr.author.login))].sort();
    const repos = [...new Set(state.data.map(pr => pr.repo))].sort();

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
}

function updateReadyParam(checked) {
    const url = new URL(window.location);
    if (checked) {
        url.searchParams.set('ready', '1');
    } else {
        url.searchParams.delete('ready');
    }
    history.replaceState(null, '', url);
}

function attachEventListeners() {
    document.querySelectorAll('.type-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.type-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            state.filters.type = btn.dataset.type;
            render();
        });
    });

    document.getElementById('author-filter').addEventListener('change', e => {
        state.filters.author = e.target.value;
        render();
    });

    document.getElementById('repo-filter').addEventListener('change', e => {
        state.filters.repo = e.target.value;
        render();
    });

    document.getElementById('ready-filter').addEventListener('change', e => {
        state.filters.readyForReview = e.target.checked;
        updateReadyParam(e.target.checked);
        render();
    });

    document.querySelectorAll('th.sortable').forEach(th => {
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
        });
    });
}

function filterPRs(prs, filters) {
    return prs.filter(pr => {
        if (filters.type === 'regular' && (pr.is_automated || /\bWIP\b/i.test(pr.title))) return false;
        if (filters.type === 'automated' && !pr.is_automated) return false;
        if (filters.type === 'wip' && !/\bWIP\b/i.test(pr.title)) return false;

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
        switch (sort.field) {
            case 'created_at': cmp = new Date(a.created_at) - new Date(b.created_at); break;
            case 'updated_at': cmp = new Date(a.updated_at) - new Date(b.updated_at); break;
            case 'reviews': cmp = a.reviews.count - b.reviews.count; break;
            case 'title': cmp = a.title.localeCompare(b.title); break;
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
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

function ageColorClass(isoDate) {
    const days = (Date.now() - new Date(isoDate)) / 86400000;
    if (days >= 30) return 'age-critical';
    if (days >= 7) return 'age-old';
    if (days >= 3) return 'age-warn';
    return '';
}

function avatarUrl(url) {
    const sep = url.includes('?') ? '&' : '?';
    return `${url}${sep}s=40`;
}

function renderCIStatus(status) {
    const title = status || 'No checks';
    return `<span class="ci-dot" title="${escapeHtml(title)}"></span>`;
}

function renderSize(size) {
    if (!size) return '';
    const cls = 'size-' + size.toLowerCase();
    return `<span class="size-badge ${cls}">${escapeHtml(size)}</span>`;
}

function renderReReview(reviews) {
    if (reviews.count === 0 || reviews.has_new_commits) {
        const title = reviews.count === 0 ? 'Not yet reviewed' : 'New commits since last review';
        return `<span class="re-review-yes" title="${title}">&#x1F440;</span>`;
    }
    return '';
}

function renderRow(pr) {
    const draftBadge = pr.is_draft ? '<span class="draft-badge">Draft</span>' : '';
    const ageCls = ageColorClass(pr.created_at);
    const ageClass = ageCls ? ` ${ageCls}` : '';
    return `<tr>
        <td class="pr-cell">
            <div class="pr-title-row">
                <img class="avatar" src="${escapeHtml(avatarUrl(pr.author.avatar_url))}" alt="${escapeHtml(pr.author.login)}" title="${escapeHtml(pr.author.login)}" width="20" height="20">
                <a href="${escapeHtml(pr.url)}" target="_blank" rel="noopener">${escapeHtml(pr.title)}</a>
                ${renderSize(pr.size)}${draftBadge}
            </div>
            <div class="pr-info-line">
                <span class="pr-repo">${escapeHtml(pr.repo)}</span>
            </div>
            <div class="pr-author-line">${escapeHtml(pr.author.login)}</div>
        </td>
        <td>${renderCIStatus(pr.ci_status)}</td>
        <td class="age-cell">${formatElapsed(pr.updated_at)}</td>
        <td class="age-cell${ageClass}">${formatElapsed(pr.created_at)}</td>
        <td>${pr.unresolved_conversations}</td>
        <td class="reviews-cell">${pr.reviews.count}</td>
        <td>${renderReReview(pr.reviews)}</td>
    </tr>`;
}

function updateSortIndicators() {
    document.querySelectorAll('th.sortable').forEach(th => {
        const existing = th.querySelector('.sort-arrow');
        if (existing) existing.remove();

        if (state.sort.field && th.dataset.sort === state.sort.field) {
            const arrow = document.createElement('span');
            arrow.className = 'sort-arrow';
            arrow.textContent = state.sort.direction === 'asc' ? '▲' : '▼';
            th.prepend(arrow);
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
    el.textContent = `Data last updated: ${date.toLocaleString()} (${formatRelativeTime(date)})`;
}

document.addEventListener('DOMContentLoaded', init);
