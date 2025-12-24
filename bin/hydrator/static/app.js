var metricExpandedState = {};
var exploreKeysExpandedState = {};
var metricLogModeState = {};

function updateExpSpacingVisibility() {
    var linearSpacing = document.getElementById('linearSpacing').checked;
    var expSpacingOption = document.getElementById('expSpacingOption');
    expSpacingOption.style.display = linearSpacing ? 'none' : 'flex';
}

document.getElementById('linearSpacing').addEventListener('change', updateExpSpacingVisibility);
updateExpSpacingVisibility();

document.getElementById('queryForm').addEventListener('submit', function(e) {
    e.preventDefault();

    var form = e.target;
    var params = new URLSearchParams();

    params.append('q', document.getElementById('query').value);

    var mergedRequested = document.getElementById('merged').checked;
    params.append('m', mergedRequested ? 'true' : 'false');

    var linearSpacing = document.getElementById('linearSpacing').checked;
    if (linearSpacing) {
        params.append('l', 'true');
    }

    var numQuantiles = document.getElementById('numQuantiles').value;
    if (numQuantiles) {
        params.append('n', numQuantiles);
    }

    if (!linearSpacing) {
        var expSpacing = document.getElementById('expSpacing').value;
        if (expSpacing) {
            params.append('e', expSpacing);
        }
    }

    var errorDiv = document.getElementById('error');
    var loadingDiv = document.getElementById('loading');
    var resultsDiv = document.getElementById('results');

    errorDiv.style.display = 'none';
    loadingDiv.style.display = 'block';
    resultsDiv.innerHTML = '';

    var submitButton = form.querySelector('button[type="submit"]');
    submitButton.disabled = true;

    fetch('/api/query?' + params.toString(), {
        method: 'GET'
    })
    .then(function(response) {
        if (!response.ok) {
            return response.text().then(function(text) {
                throw new Error(text || 'Request failed');
            });
        }
        return response.json();
    })
    .then(function(data) {
        loadingDiv.style.display = 'none';
        submitButton.disabled = false;
        displayResults(data, mergedRequested);
    })
    .catch(function(error) {
        loadingDiv.style.display = 'none';
        submitButton.disabled = false;
        errorDiv.textContent = 'Error: ' + error.message;
        errorDiv.style.display = 'block';
    });
});

function displayResults(data, mergedRequested) {
    var resultsDiv = document.getElementById('results');

    if (!data || !data.Names || data.Names.length === 0) {
        resultsDiv.innerHTML = '<p>No results found.</p>';
        return;
    }

    if (mergedRequested && data.Data.length === 1) {
        var resultItem = document.createElement('div');
        resultItem.className = 'result-item merged-result';

        var mergedDetails = document.createElement('details');
        var isExpanded = metricExpandedState['__merged__'];
        if (isExpanded !== undefined) {
            mergedDetails.open = isExpanded;
        } else {
            mergedDetails.open = true;
        }

        mergedDetails.addEventListener('toggle', function() {
            metricExpandedState['__merged__'] = mergedDetails.open;
        });

        var mergedSummary = document.createElement('summary');
        mergedSummary.className = 'result-summary';
        mergedSummary.textContent = 'Merged Result';
        mergedDetails.appendChild(mergedSummary);

        var mergedContent = document.createElement('div');
        mergedContent.className = 'result-content';

        var header = document.createElement('div');
        header.className = 'result-header';

        if (data.Names.length > 0) {
            var namesDropdown = document.createElement('details');
            namesDropdown.className = 'names-dropdown';

            var summary = document.createElement('summary');
            summary.textContent = data.Names.length + ' histogram' + (data.Names.length === 1 ? '' : 's') + ' merged';
            namesDropdown.appendChild(summary);

            var namesList = document.createElement('ul');
            namesList.className = 'names-list';
            data.Names.forEach(function(name) {
                var li = document.createElement('li');
                li.textContent = name;
                namesList.appendChild(li);
            });
            namesDropdown.appendChild(namesList);
            header.appendChild(namesDropdown);
        }

        mergedContent.appendChild(header);
        renderHistogram(mergedContent, data.Data[0], '__merged__');
        mergedDetails.appendChild(mergedContent);
        resultItem.appendChild(mergedDetails);
        resultsDiv.appendChild(resultItem);
    } else {
        data.Names.forEach(function(name, index) {
            var resultItem = document.createElement('div');
            resultItem.className = 'result-item';

            var details = document.createElement('details');
            var isExpanded = metricExpandedState[name];
            if (isExpanded !== undefined) {
                details.open = isExpanded;
            } else {
                details.open = false;
            }

            details.addEventListener('toggle', function() {
                metricExpandedState[name] = details.open;
            });

            var summary = document.createElement('summary');
            summary.className = 'result-summary';
            summary.textContent = name;
            details.appendChild(summary);

            var content = document.createElement('div');
            content.className = 'result-content';
            renderHistogram(content, data.Data[index], name);
            details.appendChild(content);

            resultItem.appendChild(details);
            resultsDiv.appendChild(resultItem);
        });
    }
}

function renderHistogram(container, histData, metricKey) {
    var stats = document.createElement('div');
    stats.className = 'stats';

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

    addStat(stats, 'Total', histData.Total);
    addStat(stats, 'Sum', formatStatValue(histData.Sum));
    addStat(stats, 'Average', formatStatValue(histData.Avg));
    addStat(stats, 'Variance', formatStatValue(histData.Vari));
    addStat(stats, 'Min', histData.Min);
    addStat(stats, 'Max', histData.Max);

    container.appendChild(stats);

    if (histData.Quantiles && histData.Quantiles.length > 0) {
        var quantilesDiv = document.createElement('div');
        quantilesDiv.className = 'quantiles';

        var quantilesHeader = document.createElement('div');
        quantilesHeader.style.display = 'flex';
        quantilesHeader.style.justifyContent = 'space-between';
        quantilesHeader.style.alignItems = 'center';
        quantilesHeader.style.marginBottom = '10px';

        var quantilesTitle = document.createElement('h3');
        quantilesTitle.textContent = 'Quantiles';
        quantilesTitle.style.margin = '0';
        quantilesHeader.appendChild(quantilesTitle);

        var logModeLabel = document.createElement('label');
        logModeLabel.className = 'option-item';
        logModeLabel.style.fontSize = '13px';
        logModeLabel.style.margin = '0';

        var logModeCheckbox = document.createElement('input');
        logModeCheckbox.type = 'checkbox';
        logModeCheckbox.checked = metricLogModeState[metricKey] || false;

        var logModeSpan = document.createElement('span');
        logModeSpan.textContent = 'Logarithmic';

        logModeLabel.appendChild(logModeCheckbox);
        logModeLabel.appendChild(logModeSpan);
        quantilesHeader.appendChild(logModeLabel);

        quantilesDiv.appendChild(quantilesHeader);

        var svgContainer = document.createElement('div');
        var useLogScale = logModeCheckbox.checked;
        var svg = createQuantilesSVG(histData.Quantiles, useLogScale);
        svgContainer.appendChild(svg);
        quantilesDiv.appendChild(svgContainer);

        logModeCheckbox.addEventListener('change', function() {
            metricLogModeState[metricKey] = logModeCheckbox.checked;
            svgContainer.innerHTML = '';
            var newSvg = createQuantilesSVG(histData.Quantiles, logModeCheckbox.checked);
            svgContainer.appendChild(newSvg);
        });

        var rawDataDetails = document.createElement('details');
        rawDataDetails.className = 'raw-data-details';

        var rawDataSummary = document.createElement('summary');
        rawDataSummary.textContent = 'Show raw data';
        rawDataDetails.appendChild(rawDataSummary);

        var table = document.createElement('table');
        var thead = document.createElement('thead');
        var headerRow = document.createElement('tr');

        var qHeader = document.createElement('th');
        qHeader.textContent = 'Quantile';
        headerRow.appendChild(qHeader);

        var vHeader = document.createElement('th');
        vHeader.textContent = 'Value';
        headerRow.appendChild(vHeader);

        thead.appendChild(headerRow);
        table.appendChild(thead);

        var tbody = document.createElement('tbody');
        var precision = findMinPrecision(histData.Quantiles);

        var ranges = [];
        var currentRange = null;

        histData.Quantiles.forEach(function(quantile, index) {
            if (!currentRange) {
                currentRange = {
                    startQ: quantile.Q,
                    endQ: quantile.Q,
                    value: quantile.V
                };
            } else if (currentRange.value === quantile.V) {
                currentRange.endQ = quantile.Q;
            } else {
                ranges.push(currentRange);
                currentRange = {
                    startQ: quantile.Q,
                    endQ: quantile.Q,
                    value: quantile.V
                };
            }
        });

        if (currentRange) {
            ranges.push(currentRange);
        }

        ranges.forEach(function(range) {
            var row = document.createElement('tr');

            var qCell = document.createElement('td');
            if (range.startQ === range.endQ) {
                qCell.textContent = range.startQ.toFixed(precision);
            } else {
                qCell.textContent = range.startQ.toFixed(precision) + ' - ' + range.endQ.toFixed(precision);
            }
            row.appendChild(qCell);

            var vCell = document.createElement('td');
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
    var stat = document.createElement('div');
    stat.className = 'stat';

    var statLabel = document.createElement('div');
    statLabel.className = 'stat-label';
    statLabel.textContent = label;
    stat.appendChild(statLabel);

    var statValue = document.createElement('div');
    statValue.className = 'stat-value';
    statValue.textContent = value;
    stat.appendChild(statValue);

    container.appendChild(stat);
}

var tabButtons = document.querySelectorAll('.tab-button');
var tabContents = document.querySelectorAll('.tab-content');

function switchToTab(targetTab) {
    tabButtons.forEach(function(btn) {
        btn.classList.remove('active');
    });
    tabContents.forEach(function(content) {
        content.classList.remove('active');
    });

    var targetButton = document.querySelector('.tab-button[data-tab="' + targetTab + '"]');
    if (targetButton) {
        targetButton.classList.add('active');
    }

    var targetContent = document.getElementById(targetTab + '-tab');
    if (targetContent) {
        targetContent.classList.add('active');
    }

    if (targetTab === 'explore') {
        loadExploreKeys();
    }

    if (targetTab === 'serverconfig' && !window.serverConfigLoaded) {
        loadServerConfig();
    }

    if (targetTab === 'config' && !window.configLoaded) {
        loadConfig();
    }
}

tabButtons.forEach(function(button) {
    button.addEventListener('click', function() {
        var targetTab = button.getAttribute('data-tab');
        window.location.hash = targetTab;
    });
});

function handleHashChange() {
    var hash = window.location.hash.substring(1);
    if (hash) {
        switchToTab(hash);
    }
}

window.addEventListener('hashchange', handleHashChange);

if (window.location.hash) {
    handleHashChange();
}

document.getElementById('refreshExplore').addEventListener('click', function() {
    loadExploreKeys();
});

function loadExploreKeys() {
    var loadingDiv = document.getElementById('exploreLoading');
    var errorDiv = document.getElementById('exploreError');
    var keysDiv = document.getElementById('exploreKeys');
    var refreshButton = document.getElementById('refreshExplore');

    loadingDiv.style.display = 'block';
    errorDiv.style.display = 'none';
    keysDiv.innerHTML = '';
    refreshButton.disabled = true;

    fetch('/api/keys', {
        method: 'GET'
    })
    .then(function(response) {
        if (!response.ok) {
            throw new Error('Failed to load keys');
        }
        return response.json();
    })
    .then(function(keys) {
        loadingDiv.style.display = 'none';
        refreshButton.disabled = false;
        displayKeys(keys);
    })
    .catch(function(error) {
        loadingDiv.style.display = 'none';
        refreshButton.disabled = false;
        errorDiv.textContent = 'Error: ' + error.message;
        errorDiv.style.display = 'block';
    });
}

function displayKeys(keys) {
    var keysDiv = document.getElementById('exploreKeys');

    if (!keys || keys.length === 0) {
        keysDiv.innerHTML = '<p>No keys found.</p>';
        return;
    }

    keys.forEach(function(key) {
        var keyItem = document.createElement('div');
        keyItem.className = 'key-item';

        var details = document.createElement('details');
        var wasExpanded = exploreKeysExpandedState[key];
        if (wasExpanded) {
            details.open = true;
        }

        var summary = document.createElement('summary');
        summary.textContent = key;
        details.appendChild(summary);

        var valuesContainer = document.createElement('div');
        valuesContainer.className = 'key-values';
        details.appendChild(valuesContainer);

        details.addEventListener('toggle', function() {
            exploreKeysExpandedState[key] = details.open;
            if (details.open) {
                loadKeyValues(key, valuesContainer);
            }
        });

        if (wasExpanded) {
            loadKeyValues(key, valuesContainer);
        }

        keyItem.appendChild(details);
        keysDiv.appendChild(keyItem);
    });
}

function loadKeyValues(key, container) {
    container.innerHTML = '<div class="key-values-loading">Loading values...</div>';

    var params = new URLSearchParams();
    params.append('key', key);

    fetch('/api/values?' + params.toString(), {
        method: 'GET'
    })
    .then(function(response) {
        if (!response.ok) {
            throw new Error('Failed to load values');
        }
        return response.json();
    })
    .then(function(values) {
        displayValues(values, container);
    })
    .catch(function(error) {
        container.innerHTML = '<div class="error">Error: ' + error.message + '</div>';
    });
}

function displayValues(values, container) {
    container.innerHTML = '';

    if (!values || values.length === 0) {
        container.innerHTML = '<p>No values found.</p>';
        return;
    }

    var valueList = document.createElement('div');
    valueList.className = 'value-list';

    values.forEach(function(value) {
        var valueItem = document.createElement('div');
        valueItem.className = 'value-item';
        valueItem.textContent = value;
        valueList.appendChild(valueItem);
    });

    container.appendChild(valueList);
}

function findMinPrecision(quantiles) {
    for (var i = 0; i <= 16; i++) {
        var seen = {};
        var hasDup = false;

        for (var j = 0; j < quantiles.length; j++) {
            var formatted = quantiles[j].Q.toFixed(i);
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
    var width = 800;
    var height = 400;
    var padding = { top: 20, right: 20, bottom: 50, left: 100 };
    var graphWidth = width - padding.left - padding.right;
    var graphHeight = height - padding.top - padding.bottom;

    var values = quantiles.map(function(q) { return q.V; });

    var allPositive = values.every(function(v) { return v > 1e-100; });
    var useRegularLog = useLogScale && allPositive;
    var useSymlog = useLogScale && !allPositive;

    function symlog(x) {
        if (x === 0) return 0;
        return (x > 0 ? 1 : -1) * Math.log10(1 + Math.abs(x));
    }

    function symlogInverse(y) {
        if (y === 0) return 0;
        return (y > 0 ? 1 : -1) * (Math.pow(10, Math.abs(y)) - 1);
    }

    var minV, maxV, rangeV;
    var transformFunc, inverseFunc;

    if (useRegularLog) {
        transformFunc = function(x) { return Math.log10(x); };
        inverseFunc = function(y) { return Math.pow(10, y); };
        var transformedValues = values.map(transformFunc);
        minV = Math.min.apply(null, transformedValues);
        maxV = Math.max.apply(null, transformedValues);
        rangeV = maxV - minV;
        if (rangeV === 0) rangeV = 1;
    } else if (useSymlog) {
        transformFunc = symlog;
        inverseFunc = symlogInverse;
        var transformedValues = values.map(symlog);
        minV = Math.min.apply(null, transformedValues);
        maxV = Math.max.apply(null, transformedValues);
        rangeV = maxV - minV;
        if (rangeV === 0) rangeV = 1;
    } else {
        transformFunc = function(x) { return x; };
        inverseFunc = function(y) { return y; };
        minV = Math.min.apply(null, values);
        maxV = Math.max.apply(null, values);
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

    var svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.setAttribute('width', width);
    svg.setAttribute('height', height);
    svg.setAttribute('class', 'quantiles-svg');

    var g = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    g.setAttribute('transform', 'translate(' + padding.left + ',' + padding.top + ')');

    var xAxis = document.createElementNS('http://www.w3.org/2000/svg', 'line');
    xAxis.setAttribute('x1', 0);
    xAxis.setAttribute('y1', graphHeight);
    xAxis.setAttribute('x2', graphWidth);
    xAxis.setAttribute('y2', graphHeight);
    xAxis.setAttribute('stroke', '#333');
    xAxis.setAttribute('stroke-width', '2');
    g.appendChild(xAxis);

    var yAxis = document.createElementNS('http://www.w3.org/2000/svg', 'line');
    yAxis.setAttribute('x1', 0);
    yAxis.setAttribute('y1', 0);
    yAxis.setAttribute('x2', 0);
    yAxis.setAttribute('y2', graphHeight);
    yAxis.setAttribute('stroke', '#333');
    yAxis.setAttribute('stroke-width', '2');
    g.appendChild(yAxis);

    for (var i = 0; i <= 10; i++) {
        var x = (i / 10) * graphWidth;
        var tick = document.createElementNS('http://www.w3.org/2000/svg', 'line');
        tick.setAttribute('x1', x);
        tick.setAttribute('y1', graphHeight);
        tick.setAttribute('x2', x);
        tick.setAttribute('y2', graphHeight + 5);
        tick.setAttribute('stroke', '#333');
        tick.setAttribute('stroke-width', '1');
        g.appendChild(tick);

        var label = document.createElementNS('http://www.w3.org/2000/svg', 'text');
        label.setAttribute('x', x);
        label.setAttribute('y', graphHeight + 20);
        label.setAttribute('text-anchor', 'middle');
        label.setAttribute('class', 'axis-label');
        label.textContent = (i / 10).toFixed(1);
        g.appendChild(label);
    }

    var yTicks = 5;
    for (var i = 0; i <= yTicks; i++) {
        var y = graphHeight - (i / yTicks) * graphHeight;
        var tick = document.createElementNS('http://www.w3.org/2000/svg', 'line');
        tick.setAttribute('x1', -5);
        tick.setAttribute('y1', y);
        tick.setAttribute('x2', 0);
        tick.setAttribute('y2', y);
        tick.setAttribute('stroke', '#333');
        tick.setAttribute('stroke-width', '1');
        g.appendChild(tick);

        var gridLine = document.createElementNS('http://www.w3.org/2000/svg', 'line');
        gridLine.setAttribute('x1', 0);
        gridLine.setAttribute('y1', y);
        gridLine.setAttribute('x2', graphWidth);
        gridLine.setAttribute('y2', y);
        gridLine.setAttribute('stroke', '#e0e0e0');
        gridLine.setAttribute('stroke-width', '1');
        g.appendChild(gridLine);

        var transformedValue = minV + (i / yTicks) * rangeV;
        var actualValue = inverseFunc(transformedValue);
        var label = document.createElementNS('http://www.w3.org/2000/svg', 'text');
        label.setAttribute('x', -10);
        label.setAttribute('y', y + 4);
        label.setAttribute('text-anchor', 'end');
        label.setAttribute('class', 'axis-label');
        label.textContent = formatValue(actualValue);
        g.appendChild(label);
    }

    var pathData = '';
    quantiles.forEach(function(quantile, index) {
        var x = quantile.Q * graphWidth;
        var valueForPlotting = transformFunc(quantile.V);
        var y = graphHeight - ((valueForPlotting - minV) / rangeV) * graphHeight;

        if (index === 0) {
            pathData += 'M ' + x + ' ' + y;
        } else {
            pathData += ' L ' + x + ' ' + y;
        }
    });

    var path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    path.setAttribute('d', pathData);
    path.setAttribute('stroke', '#007bff');
    path.setAttribute('stroke-width', '2');
    path.setAttribute('fill', 'none');
    g.appendChild(path);

    var tooltipDiv = document.createElement('div');
    tooltipDiv.className = 'svg-tooltip';
    tooltipDiv.style.display = 'none';
    document.body.appendChild(tooltipDiv);

    quantiles.forEach(function(quantile) {
        var x = quantile.Q * graphWidth;
        var valueForPlotting = transformFunc(quantile.V);
        var y = graphHeight - ((valueForPlotting - minV) / rangeV) * graphHeight;

        var circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
        circle.setAttribute('cx', x);
        circle.setAttribute('cy', y);
        circle.setAttribute('r', 4);
        circle.setAttribute('fill', '#007bff');
        circle.setAttribute('class', 'quantile-point');
        circle.style.cursor = 'pointer';

        circle.addEventListener('mouseenter', function(e) {
            var precision = findMinPrecision(quantiles);
            tooltipDiv.innerHTML = '<strong>Quantile:</strong> ' + quantile.Q.toFixed(precision) + '<br><strong>Value:</strong> ' + formatValue(quantile.V);
            tooltipDiv.style.display = 'block';
            circle.setAttribute('r', 6);
        });

        circle.addEventListener('mousemove', function(e) {
            tooltipDiv.style.left = (e.pageX + 10) + 'px';
            tooltipDiv.style.top = (e.pageY + 10) + 'px';
        });

        circle.addEventListener('mouseleave', function() {
            tooltipDiv.style.display = 'none';
            circle.setAttribute('r', 4);
        });

        g.appendChild(circle);
    });

    var xAxisLabel = document.createElementNS('http://www.w3.org/2000/svg', 'text');
    xAxisLabel.setAttribute('x', graphWidth / 2);
    xAxisLabel.setAttribute('y', graphHeight + 40);
    xAxisLabel.setAttribute('text-anchor', 'middle');
    xAxisLabel.setAttribute('class', 'axis-title');
    xAxisLabel.textContent = 'Quantile';
    g.appendChild(xAxisLabel);

    var yAxisLabel = document.createElementNS('http://www.w3.org/2000/svg', 'text');
    yAxisLabel.setAttribute('x', -graphHeight / 2);
    yAxisLabel.setAttribute('y', -60);
    yAxisLabel.setAttribute('text-anchor', 'middle');
    yAxisLabel.setAttribute('transform', 'rotate(-90)');
    yAxisLabel.setAttribute('class', 'axis-title');
    yAxisLabel.textContent = 'Value';
    g.appendChild(yAxisLabel);

    svg.appendChild(g);
    return svg;
}

document.getElementById('refreshServerConfig').addEventListener('click', function() {
    window.serverConfigLoaded = false;
    loadServerConfig();
});

document.getElementById('saveServerConfig').addEventListener('click', function() {
    saveServerConfig();
});

document.getElementById('serverConfigEditor').addEventListener('keydown', function(e) {
    if (e.shiftKey && e.key === 'Enter') {
        e.preventDefault();
        saveServerConfig();
    }
});

document.getElementById('refreshConfig').addEventListener('click', function() {
    window.configLoaded = false;
    loadConfig();
});

document.getElementById('saveConfig').addEventListener('click', function() {
    saveConfig();
});

document.getElementById('configEditor').addEventListener('keydown', function(e) {
    if (e.shiftKey && e.key === 'Enter') {
        e.preventDefault();
        saveConfig();
    }
});

function loadServerConfig() {
    var loadingDiv = document.getElementById('serverConfigLoading');
    var errorDiv = document.getElementById('serverConfigError');
    var successDiv = document.getElementById('serverConfigSuccess');
    var editor = document.getElementById('serverConfigEditor');
    var refreshButton = document.getElementById('refreshServerConfig');
    var saveButton = document.getElementById('saveServerConfig');

    loadingDiv.style.display = 'block';
    errorDiv.style.display = 'none';
    successDiv.style.display = 'none';
    editor.value = '';
    refreshButton.disabled = true;
    saveButton.disabled = true;

    fetch('/api/server-config', {
        method: 'GET'
    })
    .then(function(response) {
        if (!response.ok) {
            throw new Error('Failed to load config');
        }
        return response.json();
    })
    .then(function(config) {
        loadingDiv.style.display = 'none';
        refreshButton.disabled = false;
        saveButton.disabled = false;
        window.serverConfigLoaded = true;
        editor.value = JSON.stringify(config, null, 4);
    })
    .catch(function(error) {
        loadingDiv.style.display = 'none';
        refreshButton.disabled = false;
        saveButton.disabled = false;
        errorDiv.textContent = 'Error: ' + error.message;
        errorDiv.style.display = 'block';
    });
}

function saveServerConfig() {
    var loadingDiv = document.getElementById('serverConfigLoading');
    var errorDiv = document.getElementById('serverConfigError');
    var successDiv = document.getElementById('serverConfigSuccess');
    var editor = document.getElementById('serverConfigEditor');
    var refreshButton = document.getElementById('refreshServerConfig');
    var saveButton = document.getElementById('saveServerConfig');

    errorDiv.style.display = 'none';
    successDiv.style.display = 'none';

    var config;
    try {
        config = JSON.parse(editor.value);
    } catch (e) {
        errorDiv.textContent = 'Error: Invalid JSON - ' + e.message;
        errorDiv.style.display = 'block';
        return;
    }

    loadingDiv.style.display = 'block';
    refreshButton.disabled = true;
    saveButton.disabled = true;

    fetch('/api/server-config', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(config)
    })
    .then(function(response) {
        if (!response.ok) {
            return response.text().then(function(text) {
                throw new Error(text || 'Failed to save config');
            });
        }
        loadingDiv.style.display = 'none';
        refreshButton.disabled = false;
        saveButton.disabled = false;
        successDiv.style.display = 'block';
        setTimeout(function() {
            successDiv.style.display = 'none';
        }, 3000);
    })
    .catch(function(error) {
        loadingDiv.style.display = 'none';
        refreshButton.disabled = false;
        saveButton.disabled = false;
        errorDiv.textContent = 'Error: ' + error.message;
        errorDiv.style.display = 'block';
    });
}

function loadConfig() {
    var loadingDiv = document.getElementById('configLoading');
    var errorDiv = document.getElementById('configError');
    var successDiv = document.getElementById('configSuccess');
    var editor = document.getElementById('configEditor');
    var refreshButton = document.getElementById('refreshConfig');
    var saveButton = document.getElementById('saveConfig');

    loadingDiv.style.display = 'block';
    errorDiv.style.display = 'none';
    successDiv.style.display = 'none';
    editor.value = '';
    refreshButton.disabled = true;
    saveButton.disabled = true;

    fetch('/api/config', {
        method: 'GET'
    })
    .then(function(response) {
        if (!response.ok) {
            throw new Error('Failed to load config');
        }
        return response.json();
    })
    .then(function(config) {
        loadingDiv.style.display = 'none';
        refreshButton.disabled = false;
        saveButton.disabled = false;
        window.configLoaded = true;
        editor.value = JSON.stringify(config, null, 4);
    })
    .catch(function(error) {
        loadingDiv.style.display = 'none';
        refreshButton.disabled = false;
        saveButton.disabled = false;
        errorDiv.textContent = 'Error: ' + error.message;
        errorDiv.style.display = 'block';
    });
}

function saveConfig() {
    var loadingDiv = document.getElementById('configLoading');
    var errorDiv = document.getElementById('configError');
    var successDiv = document.getElementById('configSuccess');
    var editor = document.getElementById('configEditor');
    var refreshButton = document.getElementById('refreshConfig');
    var saveButton = document.getElementById('saveConfig');

    errorDiv.style.display = 'none';
    successDiv.style.display = 'none';

    var config;
    try {
        config = JSON.parse(editor.value);
    } catch (e) {
        errorDiv.textContent = 'Error: Invalid JSON - ' + e.message;
        errorDiv.style.display = 'block';
        return;
    }

    loadingDiv.style.display = 'block';
    refreshButton.disabled = true;
    saveButton.disabled = true;

    fetch('/api/config', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(config)
    })
    .then(function(response) {
        if (!response.ok) {
            return response.text().then(function(text) {
                throw new Error(text || 'Failed to save config');
            });
        }
        loadingDiv.style.display = 'none';
        refreshButton.disabled = false;
        saveButton.disabled = false;
        successDiv.style.display = 'block';
        setTimeout(function() {
            successDiv.style.display = 'none';
        }, 3000);
    })
    .catch(function(error) {
        loadingDiv.style.display = 'none';
        refreshButton.disabled = false;
        saveButton.disabled = false;
        errorDiv.textContent = 'Error: ' + error.message;
        errorDiv.style.display = 'block';
    });
}
