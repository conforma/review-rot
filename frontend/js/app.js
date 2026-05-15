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
        restoreFiltersFromURL();

        const response = await fetch('data.json');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const json = await response.json();
        state.data = json.pull_requests || [];
        state.generatedAt = json.generated_at;
        applyUISettings(json.ui_settings);
        populateFilters();
        initSortArrows();
        render();
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
