import type { CanvasObject, ChangeSlice, ViewSlice, AutomationSlice, ChangeScenario, ViewScenario, FlowEntry, ChapterEntry } from './types';
import type { LoadedBoard } from './loader';

export interface LayoutOptions {
  flowIndices?: Set<number>;  // filter to only these indices (from selected context)
  chapters?: ChapterEntry[];  // chapters with their flow indices for header lanes
}

const COLORS = {
  'chapter-lane': '#45475a', // Slightly lighter than swimlane
  command: '#89b4fa',     // Blue
  event: '#fab387',       // Orange/peach
  'read-model': '#94e2d5', // Teal
  endpoint: '#f5c2e7',    // Pink
  'external-event': '#f9e2af', // Pale yellow
  watcher: '#cdd6f4',     // White for automation watchers
  scenario: '#b4befe',    // Lavender
  swimlane: '#313244',    // Dark gray
  story: '#6c7086',       // Gray
  storyCommand: '#b4d0fa',  // Lighter blue (for stories referencing change slices)
  storyView: '#b8f0e8',     // Lighter teal (for stories referencing view slices)
};

const LANE_HEIGHT = 110;
const LANE_HEIGHT_WITH_IMAGE = 320;
const AUTOMATION_LANE_HEIGHT = 70;
const OBJECT_WIDTH = 140;
const OBJECT_HEIGHT = 70;
const SCENARIO_HEIGHT = 45;
const COLUMN_PADDING = 30;
const SWIMLANE_PADDING = 180;
const MOCKUP_HEIGHT = 255;
const STORY_WIDTH = 120;
const STORY_HEIGHT = 60;

function textWidth(text: string): number {
  return text.length * 8 + 20;
}

function calcSliceWidth(slice: ChangeSlice | ViewSlice | AutomationSlice): number {
  const scenarioW = Math.max(...(slice.scenarios || []).map(s => textWidth(s.name)), 0);

  if (slice.type === 'change') {
    const cs = slice as ChangeSlice;
    let triggerW = OBJECT_WIDTH;
    if (cs.trigger.kind === 'endpoint') {
      triggerW = textWidth(`${cs.trigger.endpoint.verb} ${cs.trigger.endpoint.path}`);
    } else if (cs.trigger.kind === 'externalEvent') {
      triggerW = textWidth(`⚡ ${cs.trigger.externalEvent.name}`);
    } else if (cs.trigger.kind === 'internalEvent') {
      triggerW = textWidth(`⟲ ${cs.trigger.internalEvent.eventType}`);
    }
    const commandW = textWidth(cs.name);
    const eventsW = cs.emits.reduce((sum, e) => sum + textWidth(e.type) + 5, -5);
    return Math.max(OBJECT_WIDTH, triggerW, commandW, eventsW, scenarioW);
  } else if (slice.type === 'automation') {
    const as = slice as AutomationSlice;
    let triggerW = OBJECT_WIDTH;
    if (as.trigger.kind === 'externalEvent') {
      triggerW = textWidth(`⚡ ${as.trigger.externalEvent.name}`);
    } else if (as.trigger.kind === 'internalEvent') {
      triggerW = textWidth(as.trigger.internalEvent.eventType);
    }
    const commandW = textWidth(as.name);
    const eventsW = as.emits.reduce((sum, e) => sum + textWidth(e.type) + 5, -5);
    return Math.max(OBJECT_WIDTH, triggerW, commandW, eventsW, scenarioW);
  } else {
    const vs = slice as ViewSlice;
    const endpointW = textWidth(`${vs.endpoint.verb} ${vs.endpoint.path}`);
    const rmW = textWidth(vs.readModel.name);
    return Math.max(OBJECT_WIDTH, endpointW, rmW, scenarioW);
  }
}

function calcStoryWidth(entry: FlowEntry): number {
  const nameW = textWidth(`(${entry.sliceRef})`);
  const descW = entry.description ? textWidth(entry.description) : 0;
  let instW = 0;
  if (entry.instance) {
    for (const [k, v] of Object.entries(entry.instance)) {
      const lines = JSON.stringify(v, null, 2).split('\n');
      for (const line of lines) {
        instW = Math.max(instW, textWidth(`${k}: ${line}`) * 1.3);
      }
    }
  }
  return Math.max(STORY_WIDTH, nameW, descW, instW);
}

function calcScenariosHeight(slice: ChangeSlice | ViewSlice | AutomationSlice): number {
  return (slice.scenarios?.length || 0) * (SCENARIO_HEIGHT + 5) + 10;
}

const CHAPTER_LANE_HEIGHT = 40;

export function layoutBoard(board: LoadedBoard, options?: LayoutOptions): CanvasObject[] {
  const objects: CanvasObject[] = [];
  const { manifest, slices } = board;
  const flowIndices = options?.flowIndices;
  const chapters = options?.chapters || [];

  // Use actors from manifest (definition order), filter to those with visible slices
  // Note: automation slices have no actor, but external event triggers create pseudo-actor lanes
  const visibleActors = new Set<string>();
  const actorHasImage = new Map<string, boolean>();
  const externalEventLanes = new Set<string>(); // External event sources as lanes
  for (const [name, slice] of slices.entries()) {
    const entry = manifest.flow.find(e => e.name === name);
    if (flowIndices && entry && !flowIndices.has(entry.index)) continue;
    if (slice.type === 'automation') {
      const as = slice as AutomationSlice;
      if (as.trigger.kind === 'externalEvent') {
        externalEventLanes.add(as.trigger.externalEvent.source);
      }
    } else if ('actor' in slice) {
      visibleActors.add(slice.actor);
      if (slice.image) {
        actorHasImage.set(slice.actor, true);
      }
    }
  }
  // Preserve manifest.actors order, but only include visible actors
  const actorList = manifest.actors.filter(a => visibleActors.has(a));
  // External event lanes come after actors
  const externalEventList = Array.from(externalEventLanes);

  // Calculate max scenarios height (only from visible slices)
  let maxScenariosHeight = LANE_HEIGHT;
  for (const [name, slice] of slices.entries()) {
    const entry = manifest.flow.find(e => e.name === name);
    if (flowIndices && entry && !flowIndices.has(entry.index)) continue;
    maxScenariosHeight = Math.max(maxScenariosHeight, calcScenariosHeight(slice));
  }

  // Check if any slice has internal event trigger or is automation (for automation lane)
  let hasAutomationLane = false;
  for (const [name, slice] of slices.entries()) {
    const entry = manifest.flow.find(e => e.name === name);
    if (flowIndices && entry && !flowIndices.has(entry.index)) continue;
    if (slice.type === 'automation') {
      hasAutomationLane = true;
      break;
    }
    if (slice.type === 'change' && (slice as ChangeSlice).trigger.kind === 'internalEvent') {
      hasAutomationLane = true;
      break;
    }
  }

  // Calculate Y positions - chapter header at very top, then slice names
  const chapterLaneY = 10;
  const sliceNameY = chapters.length > 0 ? chapterLaneY + CHAPTER_LANE_HEIGHT + 20 : 20;
  const actorLaneY: Record<string, number> = {};
  const actorLaneHeight: Record<string, number> = {};
  let currentY = sliceNameY + 50;
  for (const actor of actorList) {
    actorLaneY[actor] = currentY;
    const height = actorHasImage.get(actor) ? LANE_HEIGHT_WITH_IMAGE : LANE_HEIGHT;
    actorLaneHeight[actor] = height;
    currentY += height;
  }
  // External event lanes (for automation triggers)
  const externalEventLaneY: Record<string, number> = {};
  for (const extEvent of externalEventList) {
    externalEventLaneY[extEvent] = currentY;
    actorLaneY[extEvent] = currentY; // Also add to actorLaneY for compatibility
    actorLaneHeight[extEvent] = AUTOMATION_LANE_HEIGHT;
    currentY += AUTOMATION_LANE_HEIGHT;
  }
  // Automation lane (only if internal event triggers exist)
  const automationY = hasAutomationLane ? currentY + 10 : -1;
  if (hasAutomationLane) {
    currentY += AUTOMATION_LANE_HEIGHT;
  }
  const commandY = currentY + 30; // margin after actor/automation lanes
  currentY += LANE_HEIGHT;
  const eventY = currentY;
  currentY += LANE_HEIGHT;
  const scenarioY = currentY;

  // First pass: calculate column widths (slices AND stories), filtered by flowIndices
  const columnWidths: number[] = [];
  const columnEntries: FlowEntry[] = [];
  for (const entry of manifest.flow) {
    // Skip if not in selected context
    if (flowIndices && !flowIndices.has(entry.index)) continue;

    if (entry.kind === 'slice' && entry.file) {
      const slice = slices.get(entry.name);
      if (!slice) continue;
      columnWidths.push(calcSliceWidth(slice));
      columnEntries.push(entry);
    } else if (entry.kind === 'story') {
      columnWidths.push(calcStoryWidth(entry));
      columnEntries.push(entry);
    }
  }

  // Use uniform column width (max of all)
  const uniformWidth = Math.max(...columnWidths);
  for (let i = 0; i < columnWidths.length; i++) {
    columnWidths[i] = uniformWidth;
  }

  // Calculate column X positions
  const columnX: number[] = [];
  let x = SWIMLANE_PADDING;
  for (const w of columnWidths) {
    columnX.push(x);
    x += w + COLUMN_PADDING;
  }
  const totalWidth = x + SWIMLANE_PADDING;

  // Build column index mapping for chapter lanes
  const entryToColIndex = new Map<number, number>();
  for (let i = 0; i < columnEntries.length; i++) {
    entryToColIndex.set(columnEntries[i].index, i);
  }

  // Add chapter header lanes
  for (const chapter of chapters) {
    // Find first and last visible column for this chapter
    const visibleCols = chapter.flowIndices
      .filter(fi => entryToColIndex.has(fi))
      .map(fi => entryToColIndex.get(fi)!);
    if (visibleCols.length === 0) continue;

    const firstCol = Math.min(...visibleCols);
    const lastCol = Math.max(...visibleCols);
    const chapterX = columnX[firstCol];
    const chapterEndX = columnX[lastCol] + columnWidths[lastCol];

    objects.push({
      id: `chapter-${chapter.name}`,
      type: 'chapter-lane',
      x: chapterX,
      y: chapterLaneY,
      width: chapterEndX - chapterX,
      height: CHAPTER_LANE_HEIGHT,
      label: chapter.name,
      color: COLORS['chapter-lane'],
      metadata: { flowIndices: chapter.flowIndices },
    });
  }

  // Add swimlanes
  for (const actor of actorList) {
    objects.push({
      id: `lane-actor-${actor}`,
      type: 'swimlane',
      x: 0,
      y: actorLaneY[actor] - 10,
      width: totalWidth,
      height: actorLaneHeight[actor],
      label: actor,
      color: COLORS.swimlane,
    });
  }

  // External event lanes (external systems that trigger automations)
  for (const extEvent of externalEventList) {
    objects.push({
      id: `lane-external-${extEvent}`,
      type: 'swimlane',
      x: 0,
      y: externalEventLaneY[extEvent] - 10,
      width: totalWidth,
      height: AUTOMATION_LANE_HEIGHT,
      label: `⚡ ${extEvent}`,
      color: COLORS.swimlane,
    });
  }

  // Automation lane
  if (hasAutomationLane) {
    objects.push({
      id: 'lane-automation',
      type: 'swimlane',
      x: 0,
      y: automationY - 10,
      width: totalWidth,
      height: AUTOMATION_LANE_HEIGHT,
      label: 'Automations',
      color: COLORS.swimlane,
    });
  }

  for (const lane of [
    { id: 'command', label: 'Commands / Read Models', y: commandY - 10 },
    { id: 'event', label: 'Events', y: eventY - 10 },
    { id: 'scenario', label: 'Scenarios', y: scenarioY - 10, height: maxScenariosHeight },
  ]) {
    objects.push({
      id: `lane-${lane.id}`,
      type: 'swimlane',
      x: 0,
      y: lane.y,
      width: totalWidth,
      height: lane.height || LANE_HEIGHT,
      label: lane.label,
      color: COLORS.swimlane,
    });
  }

  // Second pass: create objects
  for (let colIndex = 0; colIndex < columnEntries.length; colIndex++) {
    const entry = columnEntries[colIndex];
    const colX = columnX[colIndex];
    const colW = columnWidths[colIndex];

    if (entry.kind === 'story') {
      // Find referenced slice to get actor for image placement
      const refSlice = entry.sliceRef ? slices.get(entry.sliceRef) : null;
      const storyActorY = refSlice && refSlice.type !== 'automation' && 'actor' in refSlice ? actorLaneY[refSlice.actor] : null;
      const storyActorHeight = refSlice && refSlice.type !== 'automation' && 'actor' in refSlice ? actorLaneHeight[refSlice.actor] : null;
      addStory(objects, entry, colX, colW, colIndex, sliceNameY, commandY, eventY, storyActorY, storyActorHeight, refSlice?.type);
    } else {
      const slice = slices.get(entry.name);
      if (!slice) continue;

      if (slice.type === 'automation') {
        addAutomationSlice(objects, slice as AutomationSlice, colX, colW, colIndex, sliceNameY, automationY, externalEventLaneY, commandY, eventY, scenarioY, maxScenariosHeight);
      } else if (slice.type === 'change') {
        const actorY = actorLaneY[(slice as ChangeSlice).actor];
        const laneHeight = actorLaneHeight[(slice as ChangeSlice).actor];
        addChangeSlice(objects, slice as ChangeSlice, colX, colW, colIndex, sliceNameY, actorY, laneHeight, automationY, commandY, eventY, scenarioY, maxScenariosHeight);
      } else {
        const actorY = actorLaneY[(slice as ViewSlice).actor];
        const laneHeight = actorLaneHeight[(slice as ViewSlice).actor];
        addViewSlice(objects, slice as ViewSlice, colX, colW, colIndex, sliceNameY, actorY, laneHeight, commandY, eventY, scenarioY, maxScenariosHeight);
      }
    }
  }

  return objects;
}

function addStory(
  objects: CanvasObject[],
  entry: FlowEntry,
  x: number,
  colWidth: number,
  colIndex: number,
  sliceNameY: number,
  commandY: number,
  eventY: number,
  actorY: number | null,
  actorLaneHeight: number | null,
  sliceType: 'change' | 'view' | 'automation' | undefined
): void {
  // Slice name at top (story name)
  objects.push({
    id: `slice-${colIndex}`,
    type: 'slice-name',
    x,
    y: sliceNameY,
    width: colWidth,
    height: 35,
    label: entry.name,
    color: '#6c7086', // gray for stories
    sliceIndex: colIndex,
  });

  // Mockup image if present (in actor lane of referenced slice)
  if (entry.image && actorY !== null && actorLaneHeight !== null) {
    objects.push({
      id: `mockup-${colIndex}`,
      type: 'mockup',
      x,
      y: actorY,
      width: colWidth,
      height: MOCKUP_HEIGHT,
      label: entry.image,
      color: '#45475a',
      metadata: { src: entry.image },
      sliceIndex: colIndex,
    });
  }

  // Story card in command lane
  objects.push({
    id: `story-${colIndex}`,
    type: 'story',
    x,
    y: commandY,
    width: colWidth,
    height: STORY_HEIGHT,
    label: `(${entry.sliceRef})`,
    color: sliceType === 'change' ? COLORS.storyCommand : sliceType === 'view' ? COLORS.storyView : COLORS.story,
    metadata: { sliceRef: entry.sliceRef, description: entry.description, instance: entry.instance },
    sliceIndex: colIndex,
  });

  // Emitted events (for change story steps)
  if (entry.emits && entry.emits.length > 0) {
    const eventCount = entry.emits.length;
    const eventW = Math.max(80, (colWidth - (eventCount - 1) * 5) / eventCount);
    entry.emits.forEach((emit, i) => {
      const eventType = typeof emit === 'string' ? emit : emit.type;
      const values = typeof emit === 'string' ? undefined : emit.values;
      objects.push({
        id: `story-event-${colIndex}-${i}`,
        type: 'event',
        x: x + i * (eventW + 5),
        y: eventY,
        width: eventW,
        height: OBJECT_HEIGHT,
        label: eventType,
        color: COLORS.event,
        metadata: { eventType, values, isStoryEvent: true },
        sliceIndex: colIndex,
      });
    });
  }
}

function addAutomationSlice(
  objects: CanvasObject[],
  slice: AutomationSlice,
  x: number,
  colWidth: number,
  colIndex: number,
  sliceNameY: number,
  automationY: number,
  externalEventLaneY: Record<string, number>,
  commandY: number,
  eventY: number,
  scenarioY: number,
  maxScenariosHeight: number
): void {
  // Slice border (full column)
  const borderTop = sliceNameY - 5;
  const borderBottom = scenarioY + maxScenariosHeight;
  objects.push({
    id: `slice-border-${colIndex}`,
    type: 'slice-border',
    x: x - 5,
    y: borderTop,
    width: colWidth + 10,
    height: borderBottom - borderTop,
    label: '',
    color: '#cdd6f4',
    sliceIndex: colIndex,
  });

  // Slice name at top
  objects.push({
    id: `slice-${colIndex}`,
    type: 'slice-name',
    x,
    y: sliceNameY,
    width: colWidth,
    height: 35,
    label: slice.name,
    color: '#cdd6f4',
    metadata: { devstatus: slice.devstatus },
    sliceIndex: colIndex,
  });

  // Trigger placement:
  // - Internal events: watcher in automation lane
  // - External events: external-event in its lane + watcher in automation lane
  if (slice.trigger.kind === 'internalEvent') {
    const int = slice.trigger.internalEvent;
    objects.push({
      id: `watcher-${colIndex}`,
      type: 'watcher',
      x,
      y: automationY,
      width: colWidth,
      height: 50,
      label: int.eventType,
      color: COLORS.watcher,
      metadata: { eventType: int.eventType, fields: int.fields },
      sliceIndex: colIndex,
    });
  } else if (slice.trigger.kind === 'externalEvent') {
    const ext = slice.trigger.externalEvent;
    const extLaneY = externalEventLaneY[ext.source];
    // External event in its source lane
    objects.push({
      id: `external-event-${colIndex}`,
      type: 'external-event',
      x,
      y: extLaneY,
      width: colWidth,
      height: 50,
      label: ext.name,
      color: COLORS['external-event'],
      metadata: { name: ext.name, source: ext.source, fields: ext.fields },
      sliceIndex: colIndex,
    });
    // Watcher in automation lane
    objects.push({
      id: `watcher-${colIndex}`,
      type: 'watcher',
      x,
      y: automationY,
      width: colWidth,
      height: 50,
      label: ext.name,
      color: COLORS.watcher,
      metadata: { eventType: ext.name, fields: ext.fields, isExternal: true },
      sliceIndex: colIndex,
    });
  }

  // Command - extract queried event types
  const cmdQueriedTypes: string[] = [];
  if (slice.command.query) {
    for (const q of slice.command.query) {
      if (q.types) cmdQueriedTypes.push(...q.types);
    }
  }

  objects.push({
    id: `cmd-${slice.name}`,
    type: 'command',
    x,
    y: commandY,
    width: colWidth,
    height: OBJECT_HEIGHT,
    label: slice.name,
    color: COLORS.command,
    metadata: { emitsTypes: slice.emits.map(e => e.type), queriesTypes: cmdQueriedTypes, fields: slice.command.fields, query: slice.command.query, scenarios: slice.scenarios?.length || 0 },
    sliceIndex: colIndex,
  });

  // Events - spread across column width
  const eventCount = slice.emits.length;
  const eventW = Math.max(80, (colWidth - (eventCount - 1) * 5) / eventCount);
  slice.emits.forEach((emit, i) => {
    objects.push({
      id: `event-${slice.name}-${emit.type}-${i}`,
      type: 'event',
      x: x + i * (eventW + 5),
      y: eventY,
      width: eventW,
      height: OBJECT_HEIGHT,
      label: emit.type,
      color: COLORS.event,
      metadata: { eventType: emit.type, fields: emit.fields, tags: emit.tags },
      sliceIndex: colIndex,
    });
  });

  // Scenarios
  (slice.scenarios || []).forEach((scenario, i) => {
    objects.push({
      id: `scenario-${slice.name}-${i}`,
      type: 'scenario',
      x,
      y: scenarioY + i * (SCENARIO_HEIGHT + 5),
      width: colWidth,
      height: SCENARIO_HEIGHT,
      label: scenario.name,
      color: COLORS.scenario,
      metadata: { ...formatChangeScenario(scenario), isSuccess: scenario.then.success },
      sliceIndex: colIndex,
    });
  });
}

function addChangeSlice(
  objects: CanvasObject[],
  slice: ChangeSlice,
  x: number,
  colWidth: number,
  colIndex: number,
  sliceNameY: number,
  actorY: number,
  actorLaneHeight: number,
  automationY: number,
  commandY: number,
  eventY: number,
  scenarioY: number,
  maxScenariosHeight: number
): void {
  // Slice border (full column)
  const borderTop = sliceNameY - 5;
  const borderBottom = scenarioY + maxScenariosHeight;
  objects.push({
    id: `slice-border-${colIndex}`,
    type: 'slice-border',
    x: x - 5,
    y: borderTop,
    width: colWidth + 10,
    height: borderBottom - borderTop,
    label: '',
    color: '#cdd6f4',
    sliceIndex: colIndex,
  });

  // Slice name at top
  objects.push({
    id: `slice-${colIndex}`,
    type: 'slice-name',
    x,
    y: sliceNameY,
    width: colWidth,
    height: 35,
    label: slice.name,
    color: '#cdd6f4',
    metadata: { devstatus: slice.devstatus },
    sliceIndex: colIndex,
  });

  // Mockup image if present (above endpoint in actor lane)
  const endpointY = actorY + actorLaneHeight - 55;
  if (slice.image) {
    objects.push({
      id: `mockup-${colIndex}`,
      type: 'mockup',
      x,
      y: actorY,
      width: colWidth,
      height: MOCKUP_HEIGHT,
      label: slice.image,
      color: '#45475a',
      metadata: { src: slice.image },
      sliceIndex: colIndex,
    });
  }

  // Trigger: endpoint, external event, or internal event (at bottom of actor lane)
  if (slice.trigger.kind === 'endpoint') {
    const ep = slice.trigger.endpoint;
    const endpointLabel = `${ep.verb} ${ep.path}`;
    objects.push({
      id: `endpoint-${colIndex}`,
      type: 'endpoint',
      x,
      y: endpointY,
      width: colWidth,
      height: 50,
      label: endpointLabel,
      color: COLORS.endpoint,
      metadata: { verb: ep.verb, path: ep.path, params: ep.params, body: ep.body },
      sliceIndex: colIndex,
    });
  } else if (slice.trigger.kind === 'externalEvent') {
    const ext = slice.trigger.externalEvent;
    objects.push({
      id: `external-event-${colIndex}`,
      type: 'external-event',
      x,
      y: endpointY,
      width: colWidth,
      height: 50,
      label: `⚡ ${ext.name}`,
      color: COLORS['external-event'],
      metadata: { name: ext.name, fields: ext.fields },
      sliceIndex: colIndex,
    });
  } else if (slice.trigger.kind === 'internalEvent') {
    // Watcher in automation lane
    const int = slice.trigger.internalEvent;
    objects.push({
      id: `watcher-${colIndex}`,
      type: 'watcher',
      x,
      y: automationY,
      width: colWidth,
      height: 50,
      label: int.eventType,
      color: COLORS.watcher,
      metadata: { eventType: int.eventType, fields: int.fields },
      sliceIndex: colIndex,
    });
  }

  // Command - extract queried event types
  const cmdQueriedTypes: string[] = [];
  if (slice.command.query) {
    for (const q of slice.command.query) {
      if (q.types) cmdQueriedTypes.push(...q.types);
    }
  }

  objects.push({
    id: `cmd-${slice.name}`,
    type: 'command',
    x,
    y: commandY,
    width: colWidth,
    height: OBJECT_HEIGHT,
    label: slice.name,
    color: COLORS.command,
    metadata: { emitsTypes: slice.emits.map(e => e.type), queriesTypes: cmdQueriedTypes, fields: slice.command.fields, query: slice.command.query, scenarios: slice.scenarios?.length || 0 },
    sliceIndex: colIndex,
  });

  // Events - spread across column width
  const eventCount = slice.emits.length;
  const eventW = Math.max(80, (colWidth - (eventCount - 1) * 5) / eventCount);
  slice.emits.forEach((emit, i) => {
    objects.push({
      id: `event-${slice.name}-${emit.type}-${i}`,
      type: 'event',
      x: x + i * (eventW + 5),
      y: eventY,
      width: eventW,
      height: OBJECT_HEIGHT,
      label: emit.type,
      color: COLORS.event,
      metadata: { eventType: emit.type, fields: emit.fields, tags: emit.tags },
      sliceIndex: colIndex,
    });
  });

  // Scenarios
  (slice.scenarios || []).forEach((scenario, i) => {
    objects.push({
      id: `scenario-${slice.name}-${i}`,
      type: 'scenario',
      x,
      y: scenarioY + i * (SCENARIO_HEIGHT + 5),
      width: colWidth,
      height: SCENARIO_HEIGHT,
      label: scenario.name,
      color: COLORS.scenario,
      metadata: { ...formatChangeScenario(scenario), isSuccess: scenario.then.success },
      sliceIndex: colIndex,
    });
  });
}

function addViewSlice(
  objects: CanvasObject[],
  slice: ViewSlice,
  x: number,
  colWidth: number,
  colIndex: number,
  sliceNameY: number,
  actorY: number,
  actorLaneHeight: number,
  commandY: number,
  _eventY: number,
  scenarioY: number,
  maxScenariosHeight: number
): void {
  // Slice border (full column)
  const borderTop = sliceNameY - 5;
  const borderBottom = scenarioY + maxScenariosHeight;
  objects.push({
    id: `slice-border-${colIndex}`,
    type: 'slice-border',
    x: x - 5,
    y: borderTop,
    width: colWidth + 10,
    height: borderBottom - borderTop,
    label: '',
    color: '#cdd6f4',
    sliceIndex: colIndex,
  });

  // Slice name at top
  objects.push({
    id: `slice-${colIndex}`,
    type: 'slice-name',
    x,
    y: sliceNameY,
    width: colWidth,
    height: 35,
    label: slice.name,
    color: '#cdd6f4',
    metadata: { devstatus: slice.devstatus },
    sliceIndex: colIndex,
  });

  // Mockup image if present (above endpoint in actor lane)
  const endpointY = actorY + actorLaneHeight - 55;
  if (slice.image) {
    objects.push({
      id: `mockup-${colIndex}`,
      type: 'mockup',
      x,
      y: actorY,
      width: colWidth,
      height: MOCKUP_HEIGHT,
      label: slice.image,
      color: '#45475a',
      metadata: { src: slice.image },
      sliceIndex: colIndex,
    });
  }

  const endpointLabel = `${slice.endpoint.verb} ${slice.endpoint.path}`;

  // Endpoint (at bottom of actor lane)
  objects.push({
    id: `endpoint-${colIndex}`,
    type: 'endpoint',
    x,
    y: endpointY,
    width: colWidth,
    height: 50,
    label: endpointLabel,
    color: COLORS.endpoint,
    metadata: { verb: slice.endpoint.verb, path: slice.endpoint.path, params: slice.endpoint.params, body: slice.endpoint.body },
    sliceIndex: colIndex,
  });

  // Read model - extract queried event types
  const queriedTypes: string[] = [];
  if (slice.query) {
    for (const q of slice.query) {
      if (q.types) queriedTypes.push(...q.types);
    }
  }

  objects.push({
    id: `rm-${slice.name}`,
    type: 'read-model',
    x,
    y: commandY,
    width: colWidth,
    height: OBJECT_HEIGHT,
    label: slice.readModel.name,
    color: COLORS['read-model'],
    metadata: { queriesTypes: queriedTypes, fields: slice.readModel.fields, mapping: slice.readModel.mapping, cardinality: slice.readModel.cardinality, query: slice.query },
    sliceIndex: colIndex,
  });

  // Scenarios (view scenarios are always success)
  (slice.scenarios || []).forEach((scenario, i) => {
    objects.push({
      id: `scenario-${slice.name}-${i}`,
      type: 'scenario',
      x,
      y: scenarioY + i * (SCENARIO_HEIGHT + 5),
      width: colWidth,
      height: SCENARIO_HEIGHT,
      label: scenario.name,
      color: COLORS.scenario,
      metadata: { ...formatViewScenario(scenario), isSuccess: true },
      sliceIndex: colIndex,
    });
  });
}

function formatChangeScenario(s: ChangeScenario): Record<string, unknown> {
  return {
    given: s.given,
    when: s.when,
    then: s.then,
  };
}

function formatViewScenario(s: ViewScenario): Record<string, unknown> {
  return {
    given: s.given,
    query: s.query,
    expect: s.expect,
  };
}
