let allEpics = [];
let selectedEpicKey = null;

function writeDebug(message) {
    const debug = document.getElementById("debugOutput");
    if (!debug) return;
    debug.textContent += message + "\n";
    debug.scrollTop = debug.scrollHeight;
}

function writeOutput(message) {
    const output = document.getElementById("mainOutput");
    if (!output) return;
    output.textContent += message + "\n";
    output.scrollTop = output.scrollHeight;
}

async function loadProjects() {
    const projectSelect = document.getElementById("projectSelect");

    if (!projectSelect) return;

    projectSelect.innerHTML = "<option>Loading projects...</option>";
    writeOutput("Loading projects...");

    try {
        const result = await window.go.main.App.GetProjects();

        if (result.error) {
            writeDebug("ERROR: " + result.error);
            projectSelect.innerHTML = "<option>Error loading projects</option>";
            return;
        }

        projectSelect.innerHTML = "";

        result.projects.forEach(project => {
            const option = document.createElement("option");
            option.value = project.key;
            option.textContent = `${project.key} - ${project.name}`;
            projectSelect.appendChild(option);
        });

        const backendDefault = await window.go.main.App.GetDefaultProject();
        let projectToUse = result.projects[0]?.key;

        if (backendDefault && result.projects.some(p => p.key === backendDefault)) {
            projectToUse = backendDefault;
        }

        projectSelect.value = projectToUse;
        await loadEpicsForProject(projectToUse);

    } catch (err) {
        writeDebug("JS ERROR: " + err);
    }
}

// ==========================
// Load Epics (Combo Dropdown)
// ==========================
async function loadEpicsForProject(projectKey) {

    const epicInput = document.getElementById("epicInput");
    const dropdown = document.getElementById("epicDropdown");

    if (!epicInput || !dropdown) return;

    dropdown.innerHTML = "<div class='combo-item'>Loading epics...</div>";
    writeOutput(`Fetching epics for ${projectKey}...`);

    try {
        const result = await window.go.main.App.GetEpics(projectKey);

        if (result.error) {
            dropdown.innerHTML = "<div class='combo-item'>Error loading epics</div>";
            writeDebug("ERROR: " + result.error);
            return;
        }

        allEpics = result.epics;
        renderEpicDropdown(allEpics);

    } catch (err) {
        writeDebug("JS ERROR: " + err);
    }
}

// ==========================
// Render Epic Combo Dropdown
// ==========================
function renderEpicDropdown(epics) {
    const dropdown = document.getElementById("epicDropdown");
    dropdown.innerHTML = "";

    epics.forEach(epic => {
        const item = document.createElement("div");
        item.className = "combo-item";

        const text = `${epic.key} - ${epic.title}`;
        item.textContent = text;

        item.addEventListener("click", () => {
            document.getElementById("epicInput").value = text;
            dropdown.classList.remove("open");
            selectedEpicKey = epic.key;
        });

        dropdown.appendChild(item);
    });
}

// ==========================
// Epic Input Filtering
// ==========================
document.addEventListener("DOMContentLoaded", () => {
    const epicInput = document.getElementById("epicInput");
    const dropdown = document.getElementById("epicDropdown");

    if (!epicInput || !dropdown) return;

    epicInput.addEventListener("input", () => {
        const term = epicInput.value.toLowerCase();

        const filtered = allEpics.filter(epic => {
            const text = `${epic.key} ${epic.title}`.toLowerCase();
            return text.includes(term);
        });

        renderEpicDropdown(filtered);
        dropdown.classList.add("open");
    });

    epicInput.addEventListener("focus", () => {
        dropdown.classList.add("open");
    });

    document.addEventListener("click", (e) => {
        if (!e.target.closest(".combo-box")) {
            dropdown.classList.remove("open");
        }
    });
});

// ==========================
// Load Templates (Flat Grid – No Folders)
// ==========================
async function loadTemplates() {
    const container = document.getElementById("templatesContainer");

    if (!container) return;

    container.innerHTML = "Loading templates...";
    writeOutput("Loading templates...");

    try {
        const result = await window.go.main.App.GetTemplates();

        if (result.error) {
            writeDebug("ERROR loading templates: " + result.error);
            container.innerHTML = "<i>Error loading templates</i>";
            return;
        }

        if (!result.templates || result.templates.length === 0) {
            container.innerHTML = "<i>No templates found</i>";
            return;
        }

        container.innerHTML = "";
        container.classList.add("template-grid");  // ensure 2-column layout

        result.templates.forEach(template => {
            const row = document.createElement("div");
            row.className = "template-row";

            const checkbox = document.createElement("input");
            checkbox.type = "checkbox";
            checkbox.value = template.name;
            checkbox.className = "template-checkbox";

            const label = document.createElement("label");
            label.textContent = template.name;

            // Make the entire row clickable
            row.addEventListener("click", (e) => {
                if (e.target !== checkbox) {
                    checkbox.checked = !checkbox.checked;
                }
            });

            row.appendChild(checkbox);
            row.appendChild(label);
            container.appendChild(row);
        });

    } catch (err) {
        writeDebug("JS ERROR: " + err);
    }
}

// ==========================
// Template Search Filter
// ==========================
document.addEventListener("DOMContentLoaded", () => {
    const search = document.getElementById("templateSearch");

    if (!search) return;

    search.addEventListener("input", () => {
        const term = search.value.toLowerCase();

        document.querySelectorAll(".template-row").forEach(row => {
            const text = row.innerText.toLowerCase();
            row.style.display = text.includes(term) ? "flex" : "none";
        });
    });
});

// ==========================
// Selected Templates
// ==========================
function getSelectedTemplates() {
    return Array.from(document.querySelectorAll(".template-checkbox"))
        .filter(cb => cb.checked)
        .map(cb => cb.value);
}

// ==========================
// Build Command Preview
// ==========================
function buildGojitzuCommand() {
    const project = document.getElementById("projectSelect")?.value;
    const templates = getSelectedTemplates();

    if (!project) return "ERROR: No project selected";
    if (!selectedEpicKey) return "ERROR: No epic selected";
    if (templates.length === 0) return "ERROR: No templates selected";

    let cmd = ["gojitzu", "tpl"];

    cmd.push("-p", project);
    cmd.push("-e", selectedEpicKey);

    templates.forEach(t => {
        cmd.push("-t", t);
    });

    return cmd.join(" ");
}

// ==========================
// Run Gojitzu
// ==========================
async function runGojitzu() {

    const templates = getSelectedTemplates();

    if (!selectedEpicKey) {
        alert("Please select an epic.");
        return;
    }

    if (templates.length === 0) {
        alert("Please select at least one template.");
        return;
    }

    const project = document.getElementById("projectSelect").value;
    // const args = ["-p", project, "-e", selectedEpicKey];
    const args = ["-p", project, "-e", selectedEpicKey, "--nextgen"];


    templates.forEach(t => {
        args.push("-t", t);
    });

    writeDebug("Running with args: " + JSON.stringify(args));
    writeOutput("Running gojitzu...");
    await window.go.main.App.RunGojitzuAsync(args);
}

// ==========================
// Preview Gojitzu Button
// ==========================
document.getElementById("previewBtn")?.addEventListener("click", () => {
    writeOutput("======= Command Preview =======");
    writeOutput(buildGojitzuCommand());
    writeOutput("===============================");
});

// ==========================
// Debug Panel Toggle
// ==========================
document.addEventListener("DOMContentLoaded", () => {
    const debugHeader = document.getElementById("debugHeader");
    const debugContent = document.getElementById("debugContent");

    if (!debugHeader || !debugContent) return;

    debugHeader.addEventListener("click", () => {
        const isOpen = debugContent.classList.toggle("open");
        debugHeader.textContent = isOpen ? "▼ Debug Output" : "▶ Debug Output";
    });
});

// ==========================
// Event Wiring
// ==========================
document.addEventListener("DOMContentLoaded", () => {

    const projectSelect = document.getElementById("projectSelect");

    if (projectSelect) {
        projectSelect.addEventListener("change", async () => {
            await loadEpicsForProject(projectSelect.value);
        });
    }

    const runBtn = document.getElementById("runBtn");
    if (runBtn) runBtn.addEventListener("click", runGojitzu);

    loadProjects();
    loadTemplates();
});
