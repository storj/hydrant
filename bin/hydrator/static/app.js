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
        resultItem.className = 'result-item';

        var header = document.createElement('div');
        header.className = 'result-header';

        var title = document.createElement('h2');
        title.textContent = 'Merged Result';
        header.appendChild(title);

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

        resultItem.appendChild(header);
        renderHistogram(resultItem, data.Data[0]);
        resultsDiv.appendChild(resultItem);
    } else {
        data.Names.forEach(function(name, index) {
            var resultItem = document.createElement('div');
            resultItem.className = 'result-item';

            var details = document.createElement('details');

            var summary = document.createElement('summary');
            summary.className = 'result-summary';
            summary.textContent = name;
            details.appendChild(summary);

            var content = document.createElement('div');
            content.className = 'result-content';
            renderHistogram(content, data.Data[index]);
            details.appendChild(content);

            resultItem.appendChild(details);
            resultsDiv.appendChild(resultItem);
        });
    }
}

function renderHistogram(container, histData) {
    var stats = document.createElement('div');
    stats.className = 'stats';

    addStat(stats, 'Total', histData.Total);
    addStat(stats, 'Sum', histData.Sum.toFixed(2));
    addStat(stats, 'Average', histData.Avg.toFixed(2));
    addStat(stats, 'Variance', histData.Vari.toFixed(2));
    addStat(stats, 'Min', histData.Min);
    addStat(stats, 'Max', histData.Max);

    container.appendChild(stats);

    if (histData.Quantiles && histData.Quantiles.length > 0) {
        var quantilesDiv = document.createElement('div');
        quantilesDiv.className = 'quantiles';

        var quantilesTitle = document.createElement('h3');
        quantilesTitle.textContent = 'Quantiles';
        quantilesDiv.appendChild(quantilesTitle);

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

        histData.Quantiles.forEach(function(quantile) {
            var row = document.createElement('tr');

            var qCell = document.createElement('td');
            qCell.textContent = quantile.Q.toFixed(precision);
            row.appendChild(qCell);

            var vCell = document.createElement('td');
            vCell.textContent = quantile.V;
            row.appendChild(vCell);

            tbody.appendChild(row);
        });

        table.appendChild(tbody);
        quantilesDiv.appendChild(table);
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

tabButtons.forEach(function(button) {
    button.addEventListener('click', function() {
        var targetTab = button.getAttribute('data-tab');

        tabButtons.forEach(function(btn) {
            btn.classList.remove('active');
        });
        tabContents.forEach(function(content) {
            content.classList.remove('active');
        });

        button.classList.add('active');
        document.getElementById(targetTab + '-tab').classList.add('active');

        if (targetTab === 'explore' && !window.exploreLoaded) {
            loadExploreKeys();
        }

        if (targetTab === 'config' && !window.configLoaded) {
            loadConfig();
        }
    });
});

document.getElementById('refreshExplore').addEventListener('click', function() {
    window.exploreLoaded = false;
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
        window.exploreLoaded = true;
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
        var summary = document.createElement('summary');
        summary.textContent = key;
        details.appendChild(summary);

        var valuesContainer = document.createElement('div');
        valuesContainer.className = 'key-values';
        details.appendChild(valuesContainer);

        details.addEventListener('toggle', function() {
            if (details.open) {
                loadKeyValues(key, valuesContainer);
            }
        });

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

document.getElementById('refreshConfig').addEventListener('click', function() {
    window.configLoaded = false;
    loadConfig();
});

document.getElementById('saveConfig').addEventListener('click', function() {
    saveConfig();
});

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
