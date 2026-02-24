import type { CanvasObject, Viewport, BoardManifest } from './data/types';
import { loadBoard, watchBoard, BOARD_PATH } from './data/loader';
import type { LoadedBoard } from './data/loader';
import { layoutBoard, LayoutOptions } from './data/layout';
import { SpatialIndex } from './data/SpatialIndex';
import { Renderer } from './canvas/Renderer';
import { InputHandler } from './canvas/InputHandler';
import { HitTester } from './canvas/HitTester';

const canvas = document.getElementById('canvas') as HTMLCanvasElement;
const tooltip = document.getElementById('tooltip') as HTMLDivElement;
const status = document.getElementById('status') as HTMLDivElement;
const info = document.getElementById('info') as HTMLDivElement;
const sidebar = document.getElementById('sidebar') as HTMLDivElement;
const sidebarToggle = document.getElementById('sidebar-toggle') as HTMLButtonElement;
const imageModal = document.getElementById('image-modal') as HTMLDivElement;
const modalImage = document.getElementById('modal-image') as HTMLImageElement;

const viewport: Viewport = {
    x: -20,
    y: -20,
    zoom: 1,
    width: window.innerWidth,
    height: window.innerHeight,
};

const index = new SpatialIndex();
let renderer: Renderer;
let objects: CanvasObject[] = [];

// Hidden slices state
let hiddenSlices: Set<number> = new Set();
let currentHighlightSet: Set<string> | null = null;
let hoveredSliceIndex: number | null = null;
let originalObjects: CanvasObject[] = [];
let sliceWidths: Map<number, number> = new Map(); // sliceIndex -> width+padding
let sliceOriginalX: Map<number, number> = new Map(); // sliceIndex -> original X
let mouseX = 0, mouseY = 0;

// Event type mappings for highlight on hover
let eventTypeToEvents: Map<string, string[]> = new Map();
let eventTypeToCommands: Map<string, string[]> = new Map();
let eventTypeToReadModels: Map<string, string[]> = new Map();
let eventTypeToWatchers: Map<string, string[]> = new Map();
// Automation -> consumed read model names (for automation highlighting)
let automationToConsumedReadModels: Map<string, string[]> = new Map();
// Read model name -> object IDs (for lookup)
let readModelNameToIds: Map<string, string[]> = new Map();

function buildEventTypeMappings(objs: CanvasObject[]): void {
    eventTypeToEvents = new Map();
    eventTypeToCommands = new Map();
    eventTypeToReadModels = new Map();
    eventTypeToWatchers = new Map();
    automationToConsumedReadModels = new Map();
    readModelNameToIds = new Map();

    for (const obj of objs) {
        if (obj.type === 'event' && obj.metadata?.eventType) {
            const t = obj.metadata.eventType as string;
            if (!eventTypeToEvents.has(t)) eventTypeToEvents.set(t, []);
            eventTypeToEvents.get(t)!.push(obj.id);
        }
        if (obj.type === 'command' && obj.metadata?.emitsTypes) {
            for (const t of obj.metadata.emitsTypes as string[]) {
                if (!eventTypeToCommands.has(t)) eventTypeToCommands.set(t, []);
                eventTypeToCommands.get(t)!.push(obj.id);
            }
            // Track automation consumes for commands in automation slices
            if (obj.metadata?.consumes) {
                const names = (obj.metadata.consumes as { name: string }[]).map(c => c.name);
                automationToConsumedReadModels.set(obj.id, names);
            }
        }
        if (obj.type === 'read-model') {
            // Build name -> IDs map
            const rmName = obj.label;
            if (!readModelNameToIds.has(rmName)) readModelNameToIds.set(rmName, []);
            readModelNameToIds.get(rmName)!.push(obj.id);

            if (obj.metadata?.queriesTypes) {
                for (const t of obj.metadata.queriesTypes as string[]) {
                    if (!eventTypeToReadModels.has(t)) eventTypeToReadModels.set(t, []);
                    eventTypeToReadModels.get(t)!.push(obj.id);
                }
            }
        }
        if (obj.type === 'watcher') {
            const t = obj.metadata?.eventType as string;
            if (t) {
                if (!eventTypeToWatchers.has(t)) eventTypeToWatchers.set(t, []);
                eventTypeToWatchers.get(t)!.push(obj.id);
            }
            // Track automation consumes for watchers
            if (obj.metadata?.consumes) {
                const names = (obj.metadata.consumes as { name: string }[]).map(c => c.name);
                automationToConsumedReadModels.set(obj.id, names);
            }
        }
    }
}

function computeHighlightSet(obj: CanvasObject | null): string[] | null {
    if (!obj) return null;

    const ids: string[] = [];

    if (obj.type === 'event' && obj.metadata?.eventType) {
        const eventType = obj.metadata.eventType as string;
        ids.push(...(eventTypeToEvents.get(eventType) || []));
        ids.push(...(eventTypeToCommands.get(eventType) || []));
        ids.push(...(eventTypeToReadModels.get(eventType) || []));
        ids.push(...(eventTypeToWatchers.get(eventType) || []));
    } else if (obj.type === 'watcher' && obj.metadata?.eventType) {
        // Watcher: highlight itself and all events of the same type
        const eventType = obj.metadata.eventType as string;
        ids.push(obj.id);
        ids.push(...(eventTypeToEvents.get(eventType) || []));
        // Also highlight consumed read models
        const consumedNames = automationToConsumedReadModels.get(obj.id) || [];
        for (const name of consumedNames) {
            ids.push(...(readModelNameToIds.get(name) || []));
        }
    } else if (obj.type === 'command') {
        ids.push(obj.id);
        if (Array.isArray(obj.metadata?.queriesTypes) && obj.metadata.queriesTypes.length > 0) {
            for (const t of obj.metadata.queriesTypes as string[]) {
                ids.push(...(eventTypeToEvents.get(t) || []));
            }
        }
        // Highlight consumed read models for automation commands
        const consumedNames = automationToConsumedReadModels.get(obj.id) || [];
        for (const name of consumedNames) {
            ids.push(...(readModelNameToIds.get(name) || []));
        }
    } else if (obj.type === 'read-model' && obj.metadata?.queriesTypes) {
        ids.push(obj.id);
        for (const t of obj.metadata.queriesTypes as string[]) {
            ids.push(...(eventTypeToEvents.get(t) || []));
        }
    } else if (obj.type === 'story' && obj.metadata?.sliceRef) {
        // Story: highlight like the referenced slice's command/read-model
        const sliceRef = obj.metadata.sliceRef as string;
        const refObj = objects.find(o =>
            (o.type === 'command' || o.type === 'read-model') &&
            objects.some(s => s.type === 'slice-name' && s.label === sliceRef && s.sliceIndex === o.sliceIndex)
        );
        if (refObj) {
            ids.push(obj.id); // include the story itself
            return [...ids, ...(computeHighlightSet(refObj) || [])];
        }
        return null;
    } else {
        return null;
    }

    return ids;
}

function computeSliceWidths(): void {
    sliceWidths.clear();
    sliceOriginalX.clear();
    for (const obj of originalObjects) {
        if (obj.sliceIndex !== undefined && !sliceWidths.has(obj.sliceIndex)) {
            const sliceObjs = originalObjects.filter(o => o.sliceIndex === obj.sliceIndex);
            const minX = Math.min(...sliceObjs.map(o => o.x));
            const maxX = Math.max(...sliceObjs.map(o => o.x + o.width));
            sliceWidths.set(obj.sliceIndex, maxX - minX + 30); // +30 for COLUMN_PADDING
            sliceOriginalX.set(obj.sliceIndex, minX);
        }
    }
}

function rebuildWithHidden(): void {
    const visibleSlices = Array.from(sliceWidths.keys())
        .filter(i => !hiddenSlices.has(i))
        .sort((a, b) => a - b);

    // Compute new X for each visible slice, anchoring hovered slice
    const newSliceX: Map<number, number> = new Map();
    const anchor = hoveredSliceIndex !== null && !hiddenSlices.has(hoveredSliceIndex)
        ? sliceOriginalX.get(hoveredSliceIndex)!
        : null;

    if (anchor !== null && hoveredSliceIndex !== null) {
        const hovered = hoveredSliceIndex;
        // Hovered slice stays at anchor
        newSliceX.set(hovered, anchor);

        // Before: layout right-to-left ending at anchor
        const before = visibleSlices.filter(i => i < hovered);
        let x = anchor;
        for (let i = before.length - 1; i >= 0; i--) {
            const idx = before[i];
            x -= sliceWidths.get(idx)!;
            newSliceX.set(idx, x);
        }

        // After: layout left-to-left starting after hovered
        const after = visibleSlices.filter(i => i > hovered);
        x = anchor + sliceWidths.get(hovered)!;
        for (const idx of after) {
            newSliceX.set(idx, x);
            x += sliceWidths.get(idx)!;
        }
    } else {
        // No anchor: collapse left (original behavior)
        let x = sliceOriginalX.get(visibleSlices[0]) ?? 0;
        for (const idx of visibleSlices) {
            newSliceX.set(idx, x);
            x += sliceWidths.get(idx)!;
        }
    }

    // Compute shifts
    const xShift: Map<number, number> = new Map();
    for (const [idx, newX] of newSliceX) {
        xShift.set(idx, newX - sliceOriginalX.get(idx)!);
    }

    // Compute new total width for swimlanes
    const minNewX = Math.min(...Array.from(newSliceX.values()));
    const maxIdx = visibleSlices[visibleSlices.length - 1];
    const maxNewX = newSliceX.get(maxIdx)! + sliceWidths.get(maxIdx)!;
    const newTotalWidth = maxNewX - minNewX + 360; // 180 padding each side

    objects = originalObjects
        .filter(o => o.sliceIndex === undefined || !hiddenSlices.has(o.sliceIndex))
        .map(o => {
            if (o.type === 'chapter-lane' && o.metadata?.flowIndices) {
                // Recalculate chapter bounds from visible slices
                const flowIndices = o.metadata.flowIndices as number[];
                const visibleIndices = flowIndices.filter(fi => !hiddenSlices.has(fi));
                if (visibleIndices.length === 0) return null; // hide empty chapters
                const firstX = newSliceX.get(Math.min(...visibleIndices))!;
                const lastIdx = Math.max(...visibleIndices);
                const lastX = newSliceX.get(lastIdx)! + sliceWidths.get(lastIdx)! - 30; // subtract padding
                return { ...o, x: firstX, width: lastX - firstX };
            }
            if (o.sliceIndex === undefined) {
                // Swimlanes: adjust X and width
                return { ...o, x: minNewX - 180, width: newTotalWidth };
            }
            const shift = xShift.get(o.sliceIndex) || 0;
            return { ...o, x: o.x + shift };
        })
        .filter(Boolean) as CanvasObject[];

    index.load(objects);
    buildEventTypeMappings(objects);
    renderer?.markDirty();
}

function computeDimmedSlices(): Set<number> {
    if (!currentHighlightSet) return new Set();

    // Group objects by sliceIndex
    const sliceObjects: Map<number, CanvasObject[]> = new Map();
    for (const obj of originalObjects) {
        if (obj.sliceIndex !== undefined) {
            if (!sliceObjects.has(obj.sliceIndex)) sliceObjects.set(obj.sliceIndex, []);
            sliceObjects.get(obj.sliceIndex)!.push(obj);
        }
    }

    // Find slices where ALL objects are dimmed
    const dimmed: Set<number> = new Set();
    for (const [sliceIdx, objs] of sliceObjects) {
        const allDimmed = objs.every(o => !currentHighlightSet!.has(o.id));
        if (allDimmed) dimmed.add(sliceIdx);
    }
    return dimmed;
}

function toggleHiddenSlices(): void {
    if (hiddenSlices.size > 0) {
        // Restore all
        hiddenSlices.clear();
        rebuildWithHidden();
    } else if (currentHighlightSet) {
        // Hide dimmed slices
        hiddenSlices = computeDimmedSlices();
        if (hiddenSlices.size > 0) {
            rebuildWithHidden();
        }
    }
}

function resize(): void {
    const dpr = Math.max(2, window.devicePixelRatio || 1);
    const width = window.innerWidth;
    const height = window.innerHeight;

    // Set display size (CSS)
    canvas.style.width = `${width}px`;
    canvas.style.height = `${height}px`;

    // Set actual buffer size (scaled for DPI)
    canvas.width = width * dpr;
    canvas.height = height * dpr;

    viewport.width = width;
    viewport.height = height;
    renderer?.setDPR(dpr);
    renderer?.markDirty();
}

function zoomBy(factor: number): void {
    const worldX = viewport.x + mouseX / viewport.zoom;
    const worldY = viewport.y + mouseY / viewport.zoom;
    viewport.zoom = Math.max(0.1, Math.min(5, viewport.zoom * factor));
    viewport.x = worldX - mouseX / viewport.zoom;
    viewport.y = worldY - mouseY / viewport.zoom;
    renderer?.markDirty();
    updateInfo();
}

function updateInfo(): void {
    info.textContent = `${objects.length} objects | zoom: ${(viewport.zoom * 100).toFixed(0)}%`;
}

function formatQuery(query: any[]): string {
    if (!query || query.length === 0) return '';
    return query.map(q => {
        const types = q.types?.join(', ') || '';
        const tags = q.tags?.map((t: any) => `${t.tag}={${t.param}}`).join(', ') || '';
        return `  [${types}] ${tags ? `where ${tags}` : ''}`;
    }).join('\n');
}

function formatValue(v: unknown, indent: string): string {
    if (v === null) return 'null';
    if (Array.isArray(v)) {
        if (v.length === 0) return '[]';
        const items = v.map(item => formatValue(item, indent + '  '));
        return `[${items.map(i => indent + '  ' + i).join(',\n')}\n${indent}]`;
    }
    if (typeof v === 'object') {
        return `\n${formatFields(v as Record<string, unknown>, indent + '  ')}`;
    }
    return String(v);
}

function formatFields(fields: Record<string, unknown>, indent = '  '): string {
    return Object.entries(fields).map(([k, v]) => {
        const val = formatValue(v, indent);
        if (val.startsWith('\n')) {
            return `${indent}${k}:${val}`;
        }
        return `${indent}${k}: ${val}`;
    }).join('\n');
}

function showTooltip(obj: CanvasObject, e: MouseEvent): void {
    let content = "";

    if (obj.metadata) {
        // Endpoint details
        if (obj.type === 'endpoint') {
            content = `${obj.metadata.verb} ${obj.metadata.path}`;
            if (obj.metadata.auth) {
                const auth = obj.metadata.auth as Record<string, string>;
                content += `\n\nAuth:\n${Object.entries(auth).map(([k, v]) => `  ${k}: ${v}`).join('\n')}`;
            }
            if (obj.metadata.params) {
                const params = obj.metadata.params as Record<string, string>;
                content += `\n\nPath params:\n${Object.entries(params).map(([k, v]) => `  {${k}}: ${v}`).join('\n')}`;
            }
            if (obj.metadata.body) {
                const body = obj.metadata.body as Record<string, string>;
                content += `\n\nBody:\n${Object.entries(body).map(([k, v]) => `  ${k}: ${v}`).join('\n')}`;
            }
        }
        // Command details
        else if (obj.type === 'command') {
            if (obj.metadata.fields) {
                const fields = obj.metadata.fields as Record<string, string>;
                content += `Fields:\n${Object.entries(fields).map(([k, v]) => `  ${k}: ${v}`).join('\n')}`;
            }
            if (obj.metadata.query) {
                const queryStr = formatQuery(obj.metadata.query as any[]);
                if (queryStr) content += `\n\nQuery:\n${queryStr}`;
            }
            if (obj.metadata.consumes && (obj.metadata.consumes as any[]).length > 0) {
                const consumes = obj.metadata.consumes as { name: string }[];
                content += `\n\nConsumes:\n${consumes.map(c => `  ${c.name}`).join('\n')}`;
            }
        }
        // Read model details
        else if (obj.type === 'read-model') {
            if (obj.metadata.cardinality) {
                content += ` (${obj.metadata.cardinality})`;
            }
            if (obj.metadata.query) {
                const queryStr = formatQuery(obj.metadata.query as any[]);
                if (queryStr) content += `\n\nQuery:\n${queryStr}`;
            }
            if (obj.metadata.fields) {
                const fields = obj.metadata.fields as Record<string, unknown>;
                content += `\n\nFields:\n${formatFields(fields)}`;
            }
            if (obj.metadata.mapping) {
                const mapping = obj.metadata.mapping as Record<string, string>;
                content += `\n\nMapping:\n${Object.entries(mapping).map(([k, v]) => `  ${k} <- ${v}`).join('\n')}`;
            }
        }
        // Event details
        else if (obj.type === 'event') {
            if (obj.metadata.fields) {
                const fields = obj.metadata.fields as Record<string, string>;
                content += `Fields:\n${Object.entries(fields).map(([k, v]) => `  ${k}: ${v}`).join('\n')}`;
            }
            if (obj.metadata.tags) {
                content += `\n\nTags: ${(obj.metadata.tags as string[]).join(', ')}`;
            }
        }
        // External event details
        else if (obj.type === 'external-event') {
            content = `External Event: ${obj.metadata.name}`;
            if (obj.metadata.fields) {
                const fields = obj.metadata.fields as Record<string, string>;
                content += `\n\nFields:\n${Object.entries(fields).map(([k, v]) => `  ${k}: ${v}`).join('\n')}`;
            }
        }
        // Watcher details
        else if (obj.type === 'watcher') {
            content = `Watches: ${obj.metadata.eventType}`;
            if (obj.metadata.fields) {
                const fields = obj.metadata.fields as Record<string, string>;
                content += `\n\nFields:\n${Object.entries(fields).map(([k, v]) => `  ${k}: ${v}`).join('\n')}`;
            }
            if (obj.metadata.consumes && (obj.metadata.consumes as any[]).length > 0) {
                const consumes = obj.metadata.consumes as { name: string }[];
                content += `\n\nConsumes:\n${consumes.map(c => `  ${c.name}`).join('\n')}`;
            }
        }
        // Scenario details
        else if (obj.type === 'scenario') {
            content = obj.label;
            if (Array.isArray(obj.metadata.given) && obj.metadata.given.length > 0) {
                const given = obj.metadata.given as any[];
                const givenStr = given.map(g => {
                    if (typeof g === 'string') return g;
                    const vals = g.values ? `\n${formatFields(g.values, '    ')}` : '';
                    return `${g.type}${vals}`;
                }).join('\n  ');
                content += `\n\nGiven:\n  ${givenStr}`;
            }
            if (obj.metadata.when) {
                const when = obj.metadata.when as any;
                const vals = when.values ? `\n${formatFields(when.values, '  ')}` : '';
                content += `\n\nWhen: ${when.command}${vals}`;
            }
            if (obj.metadata.query) {
                content += `\n\nQuery:\n${formatFields(obj.metadata.query as Record<string, unknown>)}`;
            }
            if (obj.metadata.then) {
                const then = obj.metadata.then as any;
                if (then.success) {
                    content += `\n\nThen: SUCCESS → ${then.events?.join(', ') || 'ok'}`;
                } else if (then.error) {
                    content += `\n\nThen: FAIL → ${then.error}`;
                }
            }
            if (obj.metadata.expect) {
                content += `\n\nExpect:\n${formatFields(obj.metadata.expect as Record<string, unknown>)}`;
            }
        }
    }

    tooltip.textContent = content;
    tooltip.style.display = 'block';
    tooltip.style.left = `${e.clientX + 12}px`;
    tooltip.style.top = `${e.clientY + 12}px`;
}

function hideTooltip(): void {
    tooltip.style.display = 'none';
}

function showImageModal(src: string): void {
    modalImage.src = src;
    imageModal.style.display = 'flex';
}

function hideImageModal(): void {
    imageModal.style.display = 'none';
    modalImage.src = '';
}

// Tree sidebar state
let expandedChapters: Set<string> = new Set();
let selectedSliceIndex: number | null = null;

// Context selection state
let selectedContext: string | null = null;
let currentBoard: LoadedBoard | null = null;

function getLayoutOptions(manifest: BoardManifest): LayoutOptions | undefined {
    if (!manifest.contexts || manifest.contexts.length === 0) return undefined;

    // Get selected context (default to first)
    const ctx = manifest.contexts.find(c => c.name === selectedContext) || manifest.contexts[0];
    if (!ctx) return undefined;

    // Collect all flow indices from this context's chapters
    const flowIndices = new Set<number>();
    for (const chap of ctx.chapters) {
        for (const fi of chap.flowIndices) {
            flowIndices.add(fi);
        }
    }

    return {
        flowIndices,
        chapters: ctx.chapters,
    };
}

function rebuildLayout(): void {
    if (!currentBoard) return;

    const options = getLayoutOptions(currentBoard.manifest);
    originalObjects = layoutBoard(currentBoard, options);
    computeSliceWidths();
    hiddenSlices.clear();
    objects = originalObjects;
    index.load(objects);
    buildEventTypeMappings(objects);
    renderer?.markDirty();
    updateInfo();
}

function buildTree(manifest: BoardManifest): void {
    sidebar.innerHTML = '';

    if (!manifest.contexts || manifest.contexts.length === 0) {
        sidebar.innerHTML = '<div style="padding: 12px; color: #6c7086;">No contexts defined</div>';
        return;
    }

    // Context tabs at top
    if (manifest.contexts.length > 1) {
        const tabsContainer = document.createElement('div');
        tabsContainer.className = 'context-tabs';
        for (const ctx of manifest.contexts) {
            const tab = document.createElement('button');
            tab.className = 'context-tab';
            if (ctx.name === selectedContext) {
                tab.classList.add('active');
            }
            tab.textContent = ctx.name;
            tab.onclick = () => {
                selectedContext = ctx.name;
                rebuildLayout();
                buildTree(manifest);
            };
            tabsContainer.appendChild(tab);
        }
        sidebar.appendChild(tabsContainer);
    }

    // Find selected context
    const ctx = manifest.contexts.find(c => c.name === selectedContext) || manifest.contexts[0];
    if (!ctx) return;

    // Show context description if present
    if (ctx.description) {
        const descEl = document.createElement('div');
        descEl.className = 'context-description';
        descEl.textContent = ctx.description;
        sidebar.appendChild(descEl);
    }

    // Chapters for selected context
    for (const chap of ctx.chapters) {
        const chapId = `${ctx.name}-${chap.name}`;
        const chapWrapper = document.createElement('div');
        chapWrapper.className = 'tree-chapter-wrapper';
        if (!expandedChapters.has(chapId)) {
            chapWrapper.classList.add('tree-collapsed');
        }

        const chapNode = document.createElement('div');
        chapNode.className = 'tree-node tree-chapter';
        chapNode.innerHTML = `<span class="tree-icon">${expandedChapters.has(chapId) ? '▼' : '▶'}</span>${chap.name}`;
        chapNode.onclick = (e) => {
            e.stopPropagation();
            if (expandedChapters.has(chapId)) {
                expandedChapters.delete(chapId);
            } else {
                expandedChapters.add(chapId);
            }
            buildTree(manifest);
        };
        chapWrapper.appendChild(chapNode);

        const chapChildren = document.createElement('div');
        chapChildren.className = 'tree-children';

        for (const flowIdx of chap.flowIndices) {
            const entry = manifest.flow[flowIdx];
            if (!entry) continue;

            const sliceNode = document.createElement('div');
            sliceNode.className = 'tree-node tree-slice';
            if (selectedSliceIndex === flowIdx) {
                sliceNode.classList.add('selected');
            }

            let badge = '';
            if (entry.kind === 'slice') {
                if (entry.type === 'change') {
                    badge = '<span class="type-badge cmd">CMD</span>';
                } else if (entry.type === 'view') {
                    badge = '<span class="type-badge view">VIEW</span>';
                }
            }

            sliceNode.innerHTML = `${badge}${entry.name}`;
            sliceNode.onclick = (e) => {
                e.stopPropagation();
                selectedSliceIndex = flowIdx;
                navigateToSlice(flowIdx, entry.name);
                buildTree(manifest);
            };
            chapChildren.appendChild(sliceNode);
        }

        chapWrapper.appendChild(chapChildren);
        sidebar.appendChild(chapWrapper);
    }
}

function navigateToSlice(_flowIdx: number, sliceName: string): void {
    // Find the slice-name object
    const target = objects.find(o => o.type === 'slice-name' && o.label === sliceName);
    if (target) {
        // Reset zoom and center viewport on target
        viewport.zoom = 1;
        viewport.x = target.x + target.width / 2 - viewport.width / 2;
        viewport.y = target.y - 50;

        // Find command or read-model and apply highlighting
        const sliceObj = objects.find(o =>
            (o.type === 'command' || o.type === 'read-model') &&
            o.sliceIndex === target.sliceIndex
        );
        if (sliceObj) {
            const highlightIds = computeHighlightSet(sliceObj);
            currentHighlightSet = highlightIds ? new Set(highlightIds) : null;
            renderer.setHighlightSet(highlightIds);
        }

        renderer.markDirty();
        updateInfo();
    }
}

function applyBoard(board: Awaited<ReturnType<typeof loadBoard>>, isReload = false): void {
    if (board.error) {
        status.textContent = `Error: ${board.error}`;
        status.style.color = '#f38ba8';
        status.style.fontSize = '16px';
        // Keep previous state, don't update objects
        return;
    }
    status.style.color = '';
    status.style.fontSize = '';

    currentBoard = board;

    // Default to first context if not set
    if (!selectedContext && board.manifest.contexts && board.manifest.contexts.length > 0) {
        selectedContext = board.manifest.contexts[0].name;
    }

    const options = getLayoutOptions(board.manifest);
    originalObjects = layoutBoard(board, options);
    computeSliceWidths();
    hiddenSlices.clear();
    objects = originalObjects;
    index.load(objects);
    buildEventTypeMappings(objects);
    renderer?.markDirty();

    status.textContent = `${board.manifest.name}${isReload ? ' — reloaded' : ''} — ${objects.length} objects`;
    updateInfo();
    buildTree(board.manifest);
}

async function init(): Promise<void> {
    resize();
    window.addEventListener('resize', resize);
    canvas.addEventListener('mousemove', (e) => {
        mouseX = e.clientX;
        mouseY = e.clientY;
    });

    // Sidebar toggle
    sidebarToggle.addEventListener('click', () => {
        sidebar.classList.toggle('open');
        sidebarToggle.textContent = sidebar.classList.contains('open') ? '✕ Close' : '☰ Tree';
    });

    // Image modal close handlers
    imageModal.addEventListener('click', hideImageModal);

    window.addEventListener('keydown', (e) => {
        // Close image modal with Escape
        if (e.key === 'Escape' && imageModal.style.display === 'flex') {
            hideImageModal();
            return;
        }
        // Toggle sidebar with 't'
        if (e.key === 't') {
            sidebar.classList.toggle('open');
            sidebarToggle.textContent = sidebar.classList.contains('open') ? '✕ Close' : '☰ Tree';
            return;
        }
        if (e.key === ' ') {
            hideTooltip();
            if (hiddenSlices.size > 0) {
                hiddenSlices.clear();
                rebuildWithHidden();
            }
        } else if (e.key === 'c') {
            toggleHiddenSlices();
        } else if (e.key === '+' || e.key === '=') {
            zoomBy(1.2);
        } else if (e.key === '-') {
            zoomBy(0.8);
        }
    });

    renderer = new Renderer(canvas, index, viewport);
    resize(); // Re-run to set DPR now that renderer exists

    new InputHandler(canvas, viewport, () => {
        renderer.markDirty();
        updateInfo();
    });

    new HitTester(canvas, index, viewport, (obj) => {
        renderer.setHovered(obj?.id ?? null);
        const highlightIds = computeHighlightSet(obj);
        currentHighlightSet = highlightIds ? new Set(highlightIds) : null;
        hoveredSliceIndex = obj?.sliceIndex ?? null;
        renderer.setHighlightSet(highlightIds);
        if (obj) {
            canvas.addEventListener('mousemove', (e) => showTooltip(obj, e), { once: true });
        } else {
            hideTooltip();
        }
    }, (obj) => {
        // Click handler - mockup images open modal
        if (obj?.type === 'mockup' && obj.metadata?.src) {
            showImageModal(`${BOARD_PATH}/${obj.metadata.src}`);
            return;
        }
        // Click handler - focus on referenced slice for stories
        if (obj?.type === 'story' && obj.metadata?.sliceRef) {
            const sliceRef = obj.metadata.sliceRef as string;
            // Find the slice-name object for the referenced slice
            const target = objects.find(o => o.type === 'slice-name' && o.label === sliceRef);
            if (target) {
                // Reset zoom and center viewport on target
                viewport.zoom = 1;
                viewport.x = target.x + target.width / 2 - viewport.width / 2;
                viewport.y = target.y - 50;

                // Find command or read-model for the slice and apply highlighting
                const sliceObj = objects.find(o =>
                    (o.type === 'command' || o.type === 'read-model') &&
                    o.sliceIndex === target.sliceIndex
                );
                if (sliceObj) {
                    const highlightIds = computeHighlightSet(sliceObj);
                    currentHighlightSet = highlightIds ? new Set(highlightIds) : null;
                    renderer.setHighlightSet(highlightIds);
                }

                renderer.markDirty();
                updateInfo();
            }
        }
    });

    renderer.start();

    try {
        const board = await loadBoard();
        applyBoard(board);
    } catch (err) {
        status.textContent = `Error: ${err}`;
        console.error(err);
    }

    // Watch for file changes
    watchBoard(async () => {
        try {
            const board = await loadBoard();
            applyBoard(board, true);
        } catch (err) {
            status.textContent = `Error: ${err}`;
            console.error(err);
        }
    });
}

init();
