// State
let currentPath = null;
let metricExpandedState = {};
let metricLogModeState = {};
let svgTooltip = null;
let knownKinds = {}; // Cache path -> kind mappings
let lastQuery = null; // Track last query to detect changes
let liveEventSource = null; // Active SSE connection
let liveAutoScroll = true; // Auto-scroll live view
let statsInterval = null; // Auto-refresh interval for stats

// Tab Management
document.querySelectorAll('.tab-button').forEach(button => {
    button.addEventListener('click', () => {
        const tabName = button.getAttribute('data-tab');
        switchToTab(tabName);
    });
});

function switchToTab(tabName) {
    // Update tab buttons
    document.querySelectorAll('.tab-button').forEach(btn => {
        btn.classList.remove('active');
    });
    document.querySelector(`[data-tab="${tabName}"]`).classList.add('active');

    // Update tab content in sidebar
    document.querySelectorAll('.sidebar-content .tab-content').forEach(content => {
        content.classList.remove('active');
    });
    document.getElementById(`${tabName}-tab`).classList.add('active');
}

// Utility Functions
function showError(elementId, message) {
    const element = document.getElementById(elementId);
    element.textContent = message;
    element.style.display = 'block';
}

function hideError(elementId) {
    document.getElementById(elementId).style.display = 'none';
}

function showLoading(elementId) {
    document.getElementById(elementId).style.display = 'block';
}

function hideLoading(elementId) {
    document.getElementById(elementId).style.display = 'none';
}

// Fetch and display tree
async function loadTree() {
    showLoading('treeLoading');
    hideError('treeError');

    try {
        const response = await fetch('/tree');
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const tree = await response.json();

        hideLoading('treeLoading');
        renderTree(tree);
    } catch (error) {
        hideLoading('treeLoading');
        showError('treeError', `Failed to load tree: ${error.message}`);
    }
}

function renderTree(tree) {
    const treeView = document.getElementById('treeView');
    treeView.innerHTML = '';

    if (!tree) {
        treeView.innerHTML = '<div class="empty-state">No submitters found</div>';
        return;
    }

    // If the root is ConfiguredSubmitter, skip it and render its child at /sub
    if (tree.kind === 'ConfiguredSubmitter' && tree.sub && tree.sub.length > 0) {
        const rootNode = createTreeNode(tree.sub[0], '/sub', true);
        treeView.appendChild(rootNode);
    } else {
        const rootNode = createTreeNode(tree, '/sub', true);
        treeView.appendChild(rootNode);
    }
}

function createTreeNode(node, path, isRoot = false) {
    const nodeDiv = document.createElement('div');
    nodeDiv.className = isRoot ? 'tree-node tree-node-root' : 'tree-node';

    const headerDiv = document.createElement('div');
    headerDiv.className = 'tree-node-header';

    // Expand/collapse indicator
    const expandSpan = document.createElement('span');
    expandSpan.className = 'tree-node-expand';
    const hasChildren = node.sub && node.sub.length > 0;
    expandSpan.textContent = hasChildren ? '▼' : '•';

    // Kind (submitter type) - clickable
    const kindSpan = document.createElement('span');
    kindSpan.className = 'tree-node-kind';
    kindSpan.textContent = node.kind;
    kindSpan.title = `Navigate to ${path}`;
    kindSpan.addEventListener('click', (e) => {
        e.stopPropagation();
        navigateToSubmitter(path, node.kind);
    });

    headerDiv.appendChild(expandSpan);
    headerDiv.appendChild(kindSpan);
    nodeDiv.appendChild(headerDiv);

    // Children
    if (hasChildren) {
        const childrenDiv = document.createElement('div');
        childrenDiv.className = 'tree-node-children';

        node.sub.forEach((child, index) => {
            // MultiSubmitter uses indexed paths (/sub/0, /sub/1, etc.)
            // Other submitters (FilterSubmitter, etc.) use /sub for single child
            let childPath;
            if (node.kind === 'MultiSubmitter') {
                childPath = `${path}/sub/${index}`;
            } else {
                childPath = `${path}/sub`;
            }
            const childNode = createTreeNode(child, childPath);
            childrenDiv.appendChild(childNode);
        });

        nodeDiv.appendChild(childrenDiv);

        // Toggle expand/collapse
        expandSpan.style.cursor = 'pointer';
        expandSpan.addEventListener('click', (e) => {
            e.stopPropagation();
            childrenDiv.classList.toggle('collapsed');
            expandSpan.textContent = childrenDiv.classList.contains('collapsed') ? '▶' : '▼';
        });
    }

    return nodeDiv;
}

// Fetch and display names
async function loadNames() {
    showLoading('namesLoading');
    hideError('namesError');

    try {
        const response = await fetch('/names');
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const names = await response.json();

        hideLoading('namesLoading');
        renderNames(names);
    } catch (error) {
        hideLoading('namesLoading');
        showError('namesError', `Failed to load names: ${error.message}`);
    }
}

function renderNames(names) {
    const namesView = document.getElementById('namesView');
    namesView.innerHTML = '';

    if (!names || Object.keys(names).length === 0) {
        namesView.innerHTML = '<div class="empty-state">No named submitters found</div>';
        return;
    }

    // Sort names alphabetically
    const sortedNames = Object.keys(names).sort();

    sortedNames.forEach(name => {
        const kind = names[name];
        const itemDiv = document.createElement('div');
        itemDiv.className = 'name-item';

        const nameLink = document.createElement('span');
        nameLink.className = 'name-link';
        nameLink.textContent = name;
        nameLink.title = `${kind} - Click to navigate`;
        nameLink.addEventListener('click', () => {
            navigateToSubmitter(`/name/${name}`, kind);
        });

        itemDiv.appendChild(nameLink);
        namesView.appendChild(itemDiv);
    });
}

// Show config in main content
async function showConfig() {
    closeLiveStream();
    const mainContent = document.getElementById('main-content');
    mainContent.innerHTML = `
        <div style="margin-bottom: 20px; padding-bottom: 15px; border-bottom: 2px solid #dee2e6;">
            <h2 style="margin: 0 0 5px 0; color: #333;">Configuration</h2>
        </div>
        <div id="configLoading" class="loading">Loading...</div>
        <div id="configError" class="error" style="display: none;"></div>
        <pre id="configContent" style="flex: 1; margin: 0; padding: 10px; font-size: 13px; overflow: auto; background: #f8f9fa; border-radius: 4px; white-space: pre-wrap; word-break: break-word;"></pre>
    `;

    try {
        const response = await fetch('/config');
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const config = await response.json();
        hideLoading('configLoading');
        document.getElementById('configContent').textContent = JSON.stringify(config, null, '\t');
    } catch (error) {
        hideLoading('configLoading');
        showError('configError', `Failed to load config: ${error.message}`);
    }
}

// Navigate to a specific submitter
function navigateToSubmitter(path, kind) {
    // Store the kind so handleHashChange can use it
    knownKinds[path] = kind;
    currentPath = path;
    window.location.hash = path;
    // Don't call showSubmitterDetail directly - let hashchange handle it
}

// Show submitter detail view
function showSubmitterDetail(path, kind) {
    closeLiveStream();
    const mainContent = document.getElementById('main-content');

    // Render based on kind
    if (kind === 'HydratorSubmitter') {
        renderHydratorInterface(mainContent, path);
    } else {
        renderLiveInterface(mainContent, path, kind);
    }
}

function stopStatsRefresh() {
    if (statsInterval) {
        clearInterval(statsInterval);
        statsInterval = null;
    }
}

// Fetch stats and render into a container (used for initial load and refresh)
async function fetchAndRenderStats(container, basePath) {
    try {
        const resp = await fetch(`${basePath}/stats`);
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
        const stats = await resp.json();

        container.innerHTML = '';
        if (!stats || stats.length === 0) {
            container.innerHTML = '<div class="empty-state">No stats available</div>';
            return;
        }

        const table = document.createElement('table');
        table.style.cssText = 'border-collapse: collapse; font-size: 14px; margin-top: 10px;';

        for (const s of stats) {
            const row = document.createElement('tr');

            const nameCell = document.createElement('td');
            nameCell.style.cssText = 'padding: 6px 20px 6px 0; color: #6c757d;';
            nameCell.textContent = s.name;
            row.appendChild(nameCell);

            const valueCell = document.createElement('td');
            valueCell.style.cssText = 'padding: 6px 0; font-weight: 500; font-variant-numeric: tabular-nums;';
            valueCell.textContent = s.value.toLocaleString();
            row.appendChild(valueCell);

            table.appendChild(row);
        }
        container.appendChild(table);
    } catch (e) {
        container.innerHTML = `<div class="error">Failed to load stats: ${e.message}</div>`;
    }
}

// Start stats view with auto-refresh every 10 seconds
function startStatsView(container, basePath) {
    stopStatsRefresh();
    container.innerHTML = '<div class="loading">Loading stats...</div>';
    fetchAndRenderStats(container, basePath);
    statsInterval = setInterval(() => fetchAndRenderStats(container, basePath), 10000);
}

// Render Hydrator query interface (with tabs: Query / Live / Stats)
function renderHydratorInterface(container, basePath) {
    container.innerHTML = `
        <div style="margin-bottom: 0; padding-bottom: 15px; border-bottom: 2px solid #dee2e6;">
            <h2 style="margin: 0 0 5px 0; color: #333;">HydratorSubmitter</h2>
            <div style="color: #6c757d; font-size: 14px; margin-bottom: 10px;">Path: ${basePath}</div>
            <div class="tabs" id="hydratorTabs" style="margin-bottom: 0; border-bottom: none;">
                <button class="tab-button active" data-htab="query">Query</button>
                <button class="tab-button" data-htab="live">Live</button>
                <button class="tab-button" data-htab="stats">Stats</button>
            </div>
        </div>
        <div id="hydratorQueryTab" style="flex: 1; display: flex; flex-direction: column; overflow: hidden;">
            <form id="queryForm" style="flex-shrink: 0; margin-bottom: 20px; margin-top: 15px;">
                <div style="margin-bottom: 15px;">
                    <label for="query" style="display: block; margin-bottom: 5px; color: #555; font-weight: 500;">Query:</label>
                    <input type="text" id="query" name="q" value="{ }" required
                           style="width: 100%; padding: 8px 12px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; font-size: 14px;" />
                </div>

                <div style="margin-bottom: 20px; padding: 12px; background-color: #f8f9fa; border-radius: 4px;">
                    <div style="font-weight: 500; margin-bottom: 8px; color: #555;">Options</div>
                    <div style="display: flex; gap: 15px; flex-wrap: wrap; align-items: center;">
                        <label style="display: flex; align-items: center; gap: 5px;">
                            <input type="checkbox" id="merged" name="m" />
                            <span>Merge results</span>
                        </label>
                        <label style="display: flex; align-items: center; gap: 5px;">
                            <input type="checkbox" id="linearSpacing" name="l" checked />
                            <span>Linear spacing</span>
                        </label>
                        <label style="display: flex; align-items: center; gap: 5px;">
                            <span>Quantiles:</span>
                            <input type="number" id="numQuantiles" name="n" value="21" min="2"
                                   style="width: 60px; padding: 4px 8px; border: 1px solid #ddd; border-radius: 4px;" />
                        </label>
                        <label style="display: flex; align-items: center; gap: 5px; display: none;" id="expSpacingOption">
                            <span>Exp spacing:</span>
                            <input type="number" id="expSpacing" name="e" value="8" min="1"
                                   style="width: 60px; padding: 4px 8px; border: 1px solid #ddd; border-radius: 4px;" />
                        </label>
                    </div>
                </div>

                <button type="submit" style="padding: 10px 20px; background-color: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 14px;">
                    Submit Query
                </button>
            </form>

            <div id="error" class="error" style="display: none;"></div>
            <div id="loading" class="loading" style="display: none;">Loading...</div>
            <div id="results" style="flex: 1; overflow-y: auto;"></div>
        </div>
        <div id="hydratorLiveTab" style="flex: 1; display: none; flex-direction: column; overflow: hidden; margin-top: 15px;"></div>
        <div id="hydratorStatsTab" style="flex: 1; display: none; flex-direction: column; overflow: hidden; margin-top: 15px;"></div>
    `;

    // Hydrator tab switching
    const hydratorTabMap = {
        query: document.getElementById('hydratorQueryTab'),
        live: document.getElementById('hydratorLiveTab'),
        stats: document.getElementById('hydratorStatsTab'),
    };

    document.querySelectorAll('#hydratorTabs .tab-button').forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.getAttribute('data-htab');
            document.querySelectorAll('#hydratorTabs .tab-button').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');

            closeLiveStream();
            for (const [key, el] of Object.entries(hydratorTabMap)) {
                el.style.display = key === tab ? 'flex' : 'none';
            }

            if (tab === 'live') {
                startLiveView(hydratorTabMap.live, basePath);
            } else if (tab === 'stats') {
                startStatsView(hydratorTabMap.stats, basePath);
            }
        });
    });

    // Query form handler
    document.getElementById('queryForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        await submitQuery(basePath);
    });

    // Toggle exp spacing based on linear spacing
    document.getElementById('linearSpacing').addEventListener('change', (e) => {
        document.getElementById('expSpacingOption').style.display = e.target.checked ? 'none' : 'flex';
    });
}

// Submit query to hydrator
async function submitQuery(basePath) {
    const form = document.getElementById('queryForm');
    const formData = new FormData(form);
    const params = new URLSearchParams();

    const queryText = formData.get('q');
    params.append('q', queryText);
    if (document.getElementById('merged').checked) params.append('m', 'true');
    if (document.getElementById('linearSpacing').checked) params.append('l', 'true');
    params.append('n', formData.get('n'));
    if (!document.getElementById('linearSpacing').checked) {
        params.append('e', formData.get('e'));
    }

    // Clear expanded state only when query text changes
    if (lastQuery !== queryText) {
        metricExpandedState = {};
        lastQuery = queryText;
    }

    showLoading('loading');
    hideError('error');
    document.getElementById('results').innerHTML = '';

    try {
        const response = await fetch(`${basePath}/query?${params.toString()}`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();

        hideLoading('loading');
        renderQueryResults(data);
    } catch (error) {
        hideLoading('loading');
        showError('error', `Failed to execute query: ${error.message}`);
    }
}

// Render query results
function renderQueryResults(data) {
    const resultsDiv = document.getElementById('results');
    resultsDiv.innerHTML = '';

    if (!data.Names || data.Names.length === 0) {
        resultsDiv.innerHTML = '<div class="empty-state">No results found</div>';
        return;
    }

    const mergedRequested = document.getElementById('merged').checked;

    if (mergedRequested && data.Data.length === 1) {
        // Merged result
        const resultItem = document.createElement('div');
        resultItem.className = 'result-item merged-result';
        resultItem.style.marginBottom = '20px';
        resultItem.style.padding = '15px';
        resultItem.style.border = '1px solid #dee2e6';
        resultItem.style.borderRadius = '4px';

        const header = document.createElement('h2');
        header.style.marginTop = '0';
        header.style.marginBottom = '15px';
        header.textContent = 'Merged Result';
        resultItem.appendChild(header);

        if (data.Names.length > 0) {
            const namesDropdown = document.createElement('details');
            namesDropdown.style.marginBottom = '15px';

            const namesSummary = document.createElement('summary');
            namesSummary.style.cursor = 'pointer';
            namesSummary.textContent = data.Names.length + ' histogram' + (data.Names.length === 1 ? '' : 's') + ' merged';
            namesDropdown.appendChild(namesSummary);

            const namesList = document.createElement('ul');
            namesList.style.marginTop = '10px';
            data.Names.forEach(name => {
                const li = document.createElement('li');
                li.textContent = name;
                namesList.appendChild(li);
            });
            namesDropdown.appendChild(namesList);
            resultItem.appendChild(namesDropdown);
        }

        renderHistogram(resultItem, data.Data[0], '__merged__');
        resultsDiv.appendChild(resultItem);
    } else {
        // Individual results
        data.Names.forEach((name, index) => {
            const resultItem = document.createElement('div');
            resultItem.className = 'result-item';
            resultItem.style.marginBottom = '15px';

            const details = document.createElement('details');
            const isExpanded = metricExpandedState[name];
            details.open = isExpanded !== undefined ? isExpanded : (index === 0);
            details.addEventListener('toggle', () => {
                metricExpandedState[name] = details.open;
            });

            const summary = document.createElement('summary');
            summary.style.cursor = 'pointer';
            summary.style.padding = '10px';
            summary.style.backgroundColor = '#f8f9fa';
            summary.style.borderRadius = '4px';
            summary.style.fontWeight = '500';
            summary.textContent = name;
            details.appendChild(summary);

            const content = document.createElement('div');
            content.style.padding = '15px';
            renderHistogram(content, data.Data[index], name);
            details.appendChild(content);

            resultItem.appendChild(details);
            resultsDiv.appendChild(resultItem);
        });
    }
}

function renderHistogram(container, histData, metricKey) {
    // Stats
    const statsDiv = document.createElement('div');
    statsDiv.style.display = 'grid';
    statsDiv.style.gridTemplateColumns = 'repeat(auto-fit, minmax(150px, 1fr))';
    statsDiv.style.gap = '10px';
    statsDiv.style.marginBottom = '20px';

    function formatStatValue(value) {
        if (Math.abs(value) < 0.001 && value !== 0) {
            return value.toExponential(2);
        } else if (Math.abs(value) < 1) {
            return value.toFixed(6).replace(/\.?0+$/, '');
        } else if (Math.abs(value) < 100) {
            return value.toFixed(4).replace(/\.?0+$/, '');
        } else {
            return value.toFixed(2);
        }
    }

    addStat(statsDiv, 'Total', histData.Total);
    addStat(statsDiv, 'Sum', formatStatValue(histData.Sum));
    addStat(statsDiv, 'Average', formatStatValue(histData.Avg));
    addStat(statsDiv, 'Variance', formatStatValue(histData.Vari));
    addStat(statsDiv, 'Min', histData.Min);
    addStat(statsDiv, 'Max', histData.Max);

    container.appendChild(statsDiv);

    if (histData.Quantiles && histData.Quantiles.length > 0) {
        const quantilesDiv = document.createElement('div');

        const quantilesHeader = document.createElement('div');
        quantilesHeader.style.display = 'flex';
        quantilesHeader.style.justifyContent = 'space-between';
        quantilesHeader.style.alignItems = 'center';
        quantilesHeader.style.marginBottom = '10px';

        const quantilesTitle = document.createElement('h3');
        quantilesTitle.textContent = 'Quantiles';
        quantilesTitle.style.margin = '0';
        quantilesHeader.appendChild(quantilesTitle);

        const logModeLabel = document.createElement('label');
        logModeLabel.style.fontSize = '13px';
        logModeLabel.style.display = 'flex';
        logModeLabel.style.alignItems = 'center';
        logModeLabel.style.gap = '5px';
        logModeLabel.style.cursor = 'pointer';

        const logModeCheckbox = document.createElement('input');
        logModeCheckbox.type = 'checkbox';
        logModeCheckbox.checked = metricLogModeState[metricKey] || false;

        const logModeSpan = document.createElement('span');
        logModeSpan.textContent = 'Logarithmic';

        logModeLabel.appendChild(logModeCheckbox);
        logModeLabel.appendChild(logModeSpan);
        quantilesHeader.appendChild(logModeLabel);

        quantilesDiv.appendChild(quantilesHeader);

        const svgContainer = document.createElement('div');
        const svg = createQuantilesSVG(histData.Quantiles, logModeCheckbox.checked);
        svgContainer.appendChild(svg);
        quantilesDiv.appendChild(svgContainer);

        logModeCheckbox.addEventListener('change', () => {
            metricLogModeState[metricKey] = logModeCheckbox.checked;
            svgContainer.innerHTML = '';
            const newSvg = createQuantilesSVG(histData.Quantiles, logModeCheckbox.checked);
            svgContainer.appendChild(newSvg);
        });

        const rawDataDetails = document.createElement('details');
        rawDataDetails.style.marginTop = '15px';

        const rawDataSummary = document.createElement('summary');
        rawDataSummary.style.cursor = 'pointer';
        rawDataSummary.textContent = 'Show raw data';
        rawDataDetails.appendChild(rawDataSummary);

        const table = document.createElement('table');
        table.style.width = '100%';
        table.style.borderCollapse = 'collapse';
        table.style.marginTop = '10px';
        table.style.fontSize = '13px';

        const thead = document.createElement('thead');
        thead.innerHTML = `
            <tr style="background-color: #f8f9fa;">
                <th style="padding: 8px; text-align: left; border-bottom: 2px solid #dee2e6;">Quantile</th>
                <th style="padding: 8px; text-align: right; border-bottom: 2px solid #dee2e6;">Value</th>
            </tr>
        `;
        table.appendChild(thead);

        const tbody = document.createElement('tbody');
        const precision = findMinPrecision(histData.Quantiles);

        // Range collapsing
        const ranges = [];
        let currentRange = null;

        histData.Quantiles.forEach(quantile => {
            if (!currentRange) {
                currentRange = { startQ: quantile.Q, endQ: quantile.Q, value: quantile.V };
            } else if (currentRange.value === quantile.V) {
                currentRange.endQ = quantile.Q;
            } else {
                ranges.push(currentRange);
                currentRange = { startQ: quantile.Q, endQ: quantile.Q, value: quantile.V };
            }
        });

        if (currentRange) {
            ranges.push(currentRange);
        }

        ranges.forEach(range => {
            const row = document.createElement('tr');
            const qCell = document.createElement('td');
            qCell.style.padding = '6px 8px';
            qCell.style.borderBottom = '1px solid #dee2e6';

            if (range.startQ === range.endQ) {
                qCell.textContent = range.startQ.toFixed(precision);
            } else {
                qCell.textContent = range.startQ.toFixed(precision) + ' - ' + range.endQ.toFixed(precision);
            }
            row.appendChild(qCell);

            const vCell = document.createElement('td');
            vCell.style.padding = '6px 8px';
            vCell.style.textAlign = 'right';
            vCell.style.borderBottom = '1px solid #dee2e6';
            vCell.textContent = range.value;
            row.appendChild(vCell);

            tbody.appendChild(row);
        });

        table.appendChild(tbody);
        rawDataDetails.appendChild(table);
        quantilesDiv.appendChild(rawDataDetails);

        container.appendChild(quantilesDiv);
    }
}

function addStat(container, label, value) {
    const stat = document.createElement('div');
    stat.style.padding = '10px';
    stat.style.backgroundColor = '#f8f9fa';
    stat.style.borderRadius = '4px';

    const statLabel = document.createElement('div');
    statLabel.style.fontSize = '12px';
    statLabel.style.color = '#6c757d';
    statLabel.style.marginBottom = '5px';
    statLabel.textContent = label;
    stat.appendChild(statLabel);

    const statValue = document.createElement('div');
    statValue.style.fontSize = '16px';
    statValue.style.fontWeight = '500';
    statValue.textContent = value;
    stat.appendChild(statValue);

    container.appendChild(stat);
}

function findMinPrecision(quantiles) {
    for (let i = 0; i <= 16; i++) {
        const seen = {};
        let hasDup = false;

        for (const quantile of quantiles) {
            const formatted = quantile.Q.toFixed(i);
            if (seen[formatted]) {
                hasDup = true;
                break;
            }
            seen[formatted] = true;
        }

        if (!hasDup) {
            return i;
        }
    }
    return 16;
}

function createQuantilesSVG(quantiles, useLogScale) {
    const width = 800;
    const height = 400;
    const padding = { top: 20, right: 20, bottom: 50, left: 100 };
    const graphWidth = width - padding.left - padding.right;
    const graphHeight = height - padding.top - padding.bottom;

    const values = quantiles.map(q => q.V);
    const allPositive = values.every(v => v > 1e-100);
    const useRegularLog = useLogScale && allPositive;
    const useSymlog = useLogScale && !allPositive;

    function symlog(x) {
        if (x === 0) return 0;
        return (x > 0 ? 1 : -1) * Math.log10(1 + Math.abs(x));
    }

    function symlogInverse(y) {
        if (y === 0) return 0;
        return (y > 0 ? 1 : -1) * (Math.pow(10, Math.abs(y)) - 1);
    }

    let minV, maxV, rangeV;
    let transformFunc, inverseFunc;

    if (useRegularLog) {
        transformFunc = x => Math.log10(x);
        inverseFunc = y => Math.pow(10, y);
        const transformedValues = values.map(transformFunc);
        minV = Math.min(...transformedValues);
        maxV = Math.max(...transformedValues);
        rangeV = maxV - minV;
        if (rangeV === 0) rangeV = 1;
    } else if (useSymlog) {
        transformFunc = symlog;
        inverseFunc = symlogInverse;
        const transformedValues = values.map(symlog);
        minV = Math.min(...transformedValues);
        maxV = Math.max(...transformedValues);
        rangeV = maxV - minV;
        if (rangeV === 0) rangeV = 1;
    } else {
        transformFunc = x => x;
        inverseFunc = y => y;
        minV = Math.min(...values);
        maxV = Math.max(...values);
        rangeV = maxV - minV;
        if (rangeV === 0) rangeV = 1;
    }

    function formatValue(value) {
        if (Math.abs(value) < 0.001 && value !== 0) {
            return value.toExponential(2);
        } else if (Math.abs(value) < 1) {
            return value.toFixed(6).replace(/\.?0+$/, '');
        } else if (Math.abs(value) < 100) {
            return value.toFixed(4).replace(/\.?0+$/, '');
        } else {
            return value.toFixed(2);
        }
    }

    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.setAttribute('width', width);
    svg.setAttribute('height', height);
    svg.style.maxWidth = '100%';
    svg.style.height = 'auto';

    const g = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    g.setAttribute('transform', `translate(${padding.left},${padding.top})`);

    // Grid lines
    const yTicks = 5;
    for (let i = 0; i <= yTicks; i++) {
        const y = graphHeight - (i / yTicks) * graphHeight;
        const gridLine = document.createElementNS('http://www.w3.org/2000/svg', 'line');
        gridLine.setAttribute('x1', 0);
        gridLine.setAttribute('y1', y);
        gridLine.setAttribute('x2', graphWidth);
        gridLine.setAttribute('y2', y);
        gridLine.setAttribute('stroke', '#e0e0e0');
        gridLine.setAttribute('stroke-width', '1');
        g.appendChild(gridLine);
    }

    // Axes
    const xAxis = document.createElementNS('http://www.w3.org/2000/svg', 'line');
    xAxis.setAttribute('x1', 0);
    xAxis.setAttribute('y1', graphHeight);
    xAxis.setAttribute('x2', graphWidth);
    xAxis.setAttribute('y2', graphHeight);
    xAxis.setAttribute('stroke', '#333');
    xAxis.setAttribute('stroke-width', '2');
    g.appendChild(xAxis);

    const yAxis = document.createElementNS('http://www.w3.org/2000/svg', 'line');
    yAxis.setAttribute('x1', 0);
    yAxis.setAttribute('y1', 0);
    yAxis.setAttribute('x2', 0);
    yAxis.setAttribute('y2', graphHeight);
    yAxis.setAttribute('stroke', '#333');
    yAxis.setAttribute('stroke-width', '2');
    g.appendChild(yAxis);

    // X-axis ticks and labels
    for (let i = 0; i <= 10; i++) {
        const x = (i / 10) * graphWidth;
        const tick = document.createElementNS('http://www.w3.org/2000/svg', 'line');
        tick.setAttribute('x1', x);
        tick.setAttribute('y1', graphHeight);
        tick.setAttribute('x2', x);
        tick.setAttribute('y2', graphHeight + 5);
        tick.setAttribute('stroke', '#333');
        tick.setAttribute('stroke-width', '1');
        g.appendChild(tick);

        const label = document.createElementNS('http://www.w3.org/2000/svg', 'text');
        label.setAttribute('x', x);
        label.setAttribute('y', graphHeight + 20);
        label.setAttribute('text-anchor', 'middle');
        label.setAttribute('font-size', '12');
        label.textContent = (i / 10).toFixed(1);
        g.appendChild(label);
    }

    // Y-axis ticks and labels
    for (let i = 0; i <= yTicks; i++) {
        const y = graphHeight - (i / yTicks) * graphHeight;
        const tick = document.createElementNS('http://www.w3.org/2000/svg', 'line');
        tick.setAttribute('x1', -5);
        tick.setAttribute('y1', y);
        tick.setAttribute('x2', 0);
        tick.setAttribute('y2', y);
        tick.setAttribute('stroke', '#333');
        tick.setAttribute('stroke-width', '1');
        g.appendChild(tick);

        const transformedValue = minV + (i / yTicks) * rangeV;
        const actualValue = inverseFunc(transformedValue);
        const label = document.createElementNS('http://www.w3.org/2000/svg', 'text');
        label.setAttribute('x', -10);
        label.setAttribute('y', y + 4);
        label.setAttribute('text-anchor', 'end');
        label.setAttribute('font-size', '12');
        label.textContent = formatValue(actualValue);
        g.appendChild(label);
    }

    // Line path
    let pathData = '';
    quantiles.forEach((quantile, index) => {
        const x = quantile.Q * graphWidth;
        const valueForPlotting = transformFunc(quantile.V);
        const y = graphHeight - ((valueForPlotting - minV) / rangeV) * graphHeight;

        if (index === 0) {
            pathData += `M ${x} ${y}`;
        } else {
            pathData += ` L ${x} ${y}`;
        }
    });

    const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    path.setAttribute('d', pathData);
    path.setAttribute('stroke', '#007bff');
    path.setAttribute('stroke-width', '2');
    path.setAttribute('fill', 'none');
    g.appendChild(path);

    // Create or reuse tooltip
    if (!svgTooltip) {
        svgTooltip = document.createElement('div');
        svgTooltip.style.position = 'fixed';
        svgTooltip.style.display = 'none';
        svgTooltip.style.backgroundColor = 'rgba(0, 0, 0, 0.8)';
        svgTooltip.style.color = 'white';
        svgTooltip.style.padding = '8px 12px';
        svgTooltip.style.borderRadius = '4px';
        svgTooltip.style.fontSize = '12px';
        svgTooltip.style.pointerEvents = 'none';
        svgTooltip.style.zIndex = '1000';
        document.body.appendChild(svgTooltip);
    }
    const tooltipDiv = svgTooltip;

    // Data points
    quantiles.forEach(quantile => {
        const x = quantile.Q * graphWidth;
        const valueForPlotting = transformFunc(quantile.V);
        const y = graphHeight - ((valueForPlotting - minV) / rangeV) * graphHeight;

        const circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
        circle.setAttribute('cx', x);
        circle.setAttribute('cy', y);
        circle.setAttribute('r', 4);
        circle.setAttribute('fill', '#007bff');
        circle.style.cursor = 'pointer';

        circle.addEventListener('mouseenter', () => {
            const precision = findMinPrecision(quantiles);
            tooltipDiv.innerHTML = `<strong>Quantile:</strong> ${quantile.Q.toFixed(precision)}<br><strong>Value:</strong> ${formatValue(quantile.V)}`;
            tooltipDiv.style.display = 'block';
            circle.setAttribute('r', 6);
        });

        circle.addEventListener('mousemove', (e) => {
            tooltipDiv.style.left = (e.pageX + 10) + 'px';
            tooltipDiv.style.top = (e.pageY + 10) + 'px';
        });

        circle.addEventListener('mouseleave', () => {
            tooltipDiv.style.display = 'none';
            circle.setAttribute('r', 4);
        });

        g.appendChild(circle);
    });

    // Axis labels
    const xAxisLabel = document.createElementNS('http://www.w3.org/2000/svg', 'text');
    xAxisLabel.setAttribute('x', graphWidth / 2);
    xAxisLabel.setAttribute('y', graphHeight + 40);
    xAxisLabel.setAttribute('text-anchor', 'middle');
    xAxisLabel.setAttribute('font-size', '14');
    xAxisLabel.setAttribute('font-weight', '500');
    xAxisLabel.textContent = 'Quantile';
    g.appendChild(xAxisLabel);

    const yAxisLabel = document.createElementNS('http://www.w3.org/2000/svg', 'text');
    yAxisLabel.setAttribute('x', -graphHeight / 2);
    yAxisLabel.setAttribute('y', -60);
    yAxisLabel.setAttribute('text-anchor', 'middle');
    yAxisLabel.setAttribute('font-size', '14');
    yAxisLabel.setAttribute('font-weight', '500');
    yAxisLabel.setAttribute('transform', 'rotate(-90)');
    yAxisLabel.textContent = 'Value';
    g.appendChild(yAxisLabel);

    svg.appendChild(g);
    return svg;
}

// Render tabbed interface (Live / Stats) for non-hydrator submitters
function renderLiveInterface(container, path, kind) {
    container.innerHTML = `
        <div style="margin-bottom: 0; padding-bottom: 15px; border-bottom: 2px solid #dee2e6;">
            <h2 style="margin: 0 0 5px 0; color: #333;">${kind}</h2>
            <div style="color: #6c757d; font-size: 14px; margin-bottom: 10px;">Path: ${path}</div>
            <div class="tabs" id="submitterTabs" style="margin-bottom: 0; border-bottom: none;">
                <button class="tab-button active" data-stab="live">Live</button>
                <button class="tab-button" data-stab="stats">Stats</button>
            </div>
        </div>
        <div id="submitterLiveTab" style="flex: 1; display: flex; flex-direction: column; overflow: hidden; margin-top: 15px;"></div>
        <div id="submitterStatsTab" style="flex: 1; display: none; flex-direction: column; overflow: hidden; margin-top: 15px;"></div>
    `;

    const tabMap = {
        live: document.getElementById('submitterLiveTab'),
        stats: document.getElementById('submitterStatsTab'),
    };

    document.querySelectorAll('#submitterTabs .tab-button').forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.getAttribute('data-stab');
            document.querySelectorAll('#submitterTabs .tab-button').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');

            closeLiveStream();
            for (const [key, el] of Object.entries(tabMap)) {
                el.style.display = key === tab ? 'flex' : 'none';
            }

            if (tab === 'live') {
                startLiveView(tabMap.live, path);
            } else if (tab === 'stats') {
                startStatsView(tabMap.stats, path);
            }
        });
    });

    // Start with Live tab active
    startLiveView(tabMap.live, path);
}

// Close any active SSE connection and stats refresh
function closeLiveStream() {
    if (liveEventSource) {
        liveEventSource.close();
        liveEventSource = null;
    }
    stopStatsRefresh();
}

// Start the live event viewer in a container
async function startLiveView(container, basePath) {
    closeLiveStream();
    liveAutoScroll = true;

    container.innerHTML = `
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; flex-shrink: 0;">
            <span style="font-weight: 500; color: #555;">Live Events</span>
        </div>
        <div style="flex-shrink: 0; margin-bottom: 8px;">
            <input type="text" id="liveFilter" class="live-filter" placeholder="Filter events..." />
        </div>
        <div id="liveEventsWrap" style="flex: 1; position: relative; overflow: hidden; border-radius: 4px;">
            <div id="liveEvents" style="height: 100%; overflow-y: auto; font-family: monospace; font-size: 13px; background: #1e1e1e; color: #d4d4d4; padding: 4px;"></div>
            <button id="resumeScroll" class="resume-scroll-btn" style="display: none;">Resume auto-scroll</button>
        </div>
        <div id="liveStatus" style="flex-shrink: 0; padding: 5px 0; font-size: 12px; color: #6c757d;"></div>
    `;

    const eventsDiv = document.getElementById('liveEvents');
    const statusDiv = document.getElementById('liveStatus');
    const filterInput = document.getElementById('liveFilter');
    const resumeBtn = document.getElementById('resumeScroll');

    let filterText = '';
    let totalCount = 0;
    let rateTimestamps = []; // timestamps of recent events for rate calc

    function updateStatus(extra) {
        // Calculate rate: events in the last 5 seconds
        const now = Date.now();
        rateTimestamps = rateTimestamps.filter(t => now - t < 5000);
        const rate = Math.round(rateTimestamps.length / 5);
        const countStr = totalCount.toLocaleString();
        const rateStr = rate > 0 ? `, ~${rate}/s` : '';
        statusDiv.textContent = extra || `Streaming... (${countStr} events${rateStr})`;
    }

    // Detect user scrolling away from the bottom to pause auto-scroll
    let scrollIgnore = false;
    eventsDiv.addEventListener('scroll', () => {
        if (scrollIgnore) return;
        const atBottom = eventsDiv.scrollHeight - eventsDiv.scrollTop - eventsDiv.clientHeight < 40;
        if (atBottom && !liveAutoScroll) {
            liveAutoScroll = true;
            resumeBtn.style.display = 'none';
        } else if (!atBottom && liveAutoScroll) {
            liveAutoScroll = false;
            resumeBtn.style.display = '';
        }
    });

    resumeBtn.addEventListener('click', () => {
        liveAutoScroll = true;
        resumeBtn.style.display = 'none';
        eventsDiv.scrollTop = eventsDiv.scrollHeight;
    });

    // Filter handler
    filterInput.addEventListener('input', () => {
        filterText = filterInput.value.toLowerCase();
        for (const child of eventsDiv.children) {
            if (!filterText || child.dataset.search.includes(filterText)) {
                child.style.display = '';
            } else {
                child.style.display = 'none';
            }
        }
    });

    // Fetch initial snapshot
    try {
        const resp = await fetch(`${basePath}/live`);
        if (resp.ok) {
            const events = await resp.json();
            if (events && events.length > 0) {
                for (const ev of events) {
                    appendLiveEvent(eventsDiv, ev, filterText);
                    totalCount++;
                }
                updateStatus(`Loaded ${events.length} recent events. Streaming...`);
            } else {
                updateStatus('Waiting for events...');
            }
        }
    } catch (e) {
        updateStatus('Failed to load initial events.');
    }

    // Open SSE stream
    liveEventSource = new EventSource(`${basePath}/live?watch=1`);

    liveEventSource.onmessage = (e) => {
        try {
            const ev = JSON.parse(e.data);
            appendLiveEvent(eventsDiv, ev, filterText);
            totalCount++;
            rateTimestamps.push(Date.now());
            updateStatus();

            // Cap displayed events to prevent memory issues
            while (eventsDiv.children.length > 2000) {
                eventsDiv.removeChild(eventsDiv.firstChild);
            }
        } catch (err) {
            // ignore parse errors
        }
    };

    liveEventSource.onerror = () => {
        updateStatus('Connection lost. Reconnecting...');
    };
}

// Classify an event based on its annotations
function classifyEvent(ev) {
    const map = {};
    for (const ann of ev) {
        map[ann.key] = ann.value;
    }
    if ('duration' in map) {
        return { type: 'span', map };
    }
    if ('message' in map) {
        return { type: 'log', map };
    }
    return { type: 'other', map };
}

// Format a timestamp value as HH:MM:SS.mmm
function formatTimestamp(ts) {
    if (!ts) return '';
    // Try parsing as a date string or unix timestamp
    const d = new Date(ts);
    if (!isNaN(d.getTime())) {
        return d.toTimeString().slice(0, 8) + '.' + String(d.getMilliseconds()).padStart(3, '0');
    }
    // If it's already formatted, return as-is but truncate
    return ts.length > 12 ? ts.slice(0, 12) : ts;
}

// Build summary text for an event
function buildSummary(info, ev) {
    if (info.type === 'span') {
        const name = info.map['name'] || info.map['span'] || '';
        const dur = info.map['duration'] || '';
        const success = info.map['success'];
        const err = info.map['error'];
        let indicator = '';
        if (err && err !== '' && err !== 'false') {
            indicator = `<span class="event-error"> err</span>`;
        } else if (success === 'true') {
            indicator = `<span class="event-success"> ok</span>`;
        } else if (success === 'false') {
            indicator = `<span class="event-error"> fail</span>`;
        }
        return `${escapeHtml(name)} <span style="color: #6c757d;">${escapeHtml(dur)}</span>${indicator}`;
    }
    if (info.type === 'log') {
        return escapeHtml(info.map['message'] || '');
    }
    // Other: first 3 key=value pairs (excluding timestamp)
    const pairs = ev.filter(a => a.key !== 'timestamp').slice(0, 3);
    return pairs.map(a => `<span style="color: #9cdcfe;">${escapeHtml(a.key)}</span>=<span style="color: #ce9178;">${escapeHtml(a.value)}</span>`).join(' ');
}

// Serialize event text for filter matching
function eventSearchText(ev) {
    return ev.map(a => a.key + '=' + a.value).join(' ').toLowerCase();
}

// Append a single event to the live view
function appendLiveEvent(container, ev, filterText) {
    const info = classifyEvent(ev);

    const row = document.createElement('div');
    row.className = 'live-event event-' + info.type;

    // Store search text for filtering
    const searchText = eventSearchText(ev);
    row.dataset.search = searchText;

    // Apply filter if active
    if (filterText && !searchText.includes(filterText)) {
        row.style.display = 'none';
    }

    // Summary line
    const summary = document.createElement('div');
    summary.className = 'live-event-summary';

    const ts = info.map['timestamp'] ? formatTimestamp(info.map['timestamp']) : '';
    const badgeClass = info.type === 'span' ? 'event-badge-span' : info.type === 'log' ? 'event-badge-log' : 'event-badge-evt';
    const badgeLabel = info.type === 'span' ? 'SPAN' : info.type === 'log' ? 'LOG' : 'EVT';

    summary.innerHTML = `<span class="event-timestamp">${escapeHtml(ts)}</span><span class="event-badge ${badgeClass}">${badgeLabel}</span><span class="event-summary-text">${buildSummary(info, ev)}</span>`;

    summary.addEventListener('click', () => {
        row.classList.toggle('expanded');
    });

    row.appendChild(summary);

    // Detail section (hidden by default)
    const details = document.createElement('div');
    details.className = 'live-event-details';

    for (const ann of ev) {
        const line = document.createElement('div');
        line.className = 'detail-line';
        line.innerHTML = `<span style="color: #9cdcfe;">${escapeHtml(ann.key)}</span><span style="color: #d4d4d4;">=</span><span style="color: #ce9178;">${escapeHtml(ann.value)}</span>`;
        details.appendChild(line);
    }

    row.appendChild(details);
    container.appendChild(row);

    if (liveAutoScroll) {
        container.scrollTop = container.scrollHeight;
    }
}

function escapeHtml(s) {
    const div = document.createElement('div');
    div.textContent = s;
    return div.innerHTML;
}

// Handle hash changes (back/forward buttons)
window.addEventListener('hashchange', () => {
    handleHashChange();
});

async function handleHashChange() {
    const hash = window.location.hash.slice(1); // Remove the #

    if (!hash) {
        // Back to root - show empty state
        closeLiveStream();
        document.getElementById('main-content').innerHTML = '<div class="empty-state">Select a submitter from the left to view details</div>';
        currentPath = null;
        return;
    }

    // Config page
    if (hash === '/config') {
        currentPath = hash;
        showConfig();
        return;
    }

    // Check if we already know the kind
    if (knownKinds[hash]) {
        currentPath = hash;
        showSubmitterDetail(hash, knownKinds[hash]);
        return;
    }

    // Otherwise, fetch the tree to determine the kind
    try {
        const response = await fetch(`${hash}/tree`);
        if (response.ok) {
            const data = await response.json();
            knownKinds[hash] = data.kind; // Cache it
            currentPath = hash;
            showSubmitterDetail(hash, data.kind);
        } else {
            document.getElementById('main-content').innerHTML = '<div class="empty-state">Failed to load submitter</div>';
        }
    } catch (error) {
        document.getElementById('main-content').innerHTML = '<div class="empty-state">Error loading submitter</div>';
    }
}

// Initialize the page
document.addEventListener('DOMContentLoaded', async () => {
    await Promise.all([loadTree(), loadNames()]);

    // Check if there's a hash on initial load
    const hash = window.location.hash.slice(1);
    if (hash) {
        await handleHashChange();
    }
});
