import type { CanvasObject, ChangeSlice, ViewSlice, ChangeScenario, ViewScenario, FlowEntry } from './types';
import type { LoadedBoard } from './loader';

const COLORS = {
  command: '#89b4fa',     // Blue
  event: '#fab387',       // Orange/peach
  'read-model': '#94e2d5', // Teal
  endpoint: '#f5c2e7',    // Pink
  'external-event': '#f9e2af', // Pale yellow
  scenario: '#b4befe',    // Lavender
  swimlane: '#313244',    // Dark gray
  story: '#6c7086',       // Gray
};

const LANE_HEIGHT = 110;
const LANE_HEIGHT_WITH_IMAGE = 320;
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

function calcSliceWidth(slice: ChangeSlice | ViewSlice): number {
  const scenarioW = Math.max(...(slice.scenarios || []).map(s => textWidth(s.name)), 0);

  if (slice.type === 'change') {
    const cs = slice as ChangeSlice;
    let triggerW = OBJECT_WIDTH;
    if (cs.trigger.kind === 'endpoint') {
      triggerW = textWidth(`${cs.trigger.endpoint.verb} ${cs.trigger.endpoint.path}`);
    } else {
      triggerW = textWidth(`⚡ ${cs.trigger.externalEvent.name}`);
    }
    const commandW = textWidth(cs.name);
    const eventsW = cs.emits.reduce((sum, e) => sum + textWidth(e.type) + 5, -5);
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

function calcScenariosHeight(slice: ChangeSlice | ViewSlice): number {
  return (slice.scenarios?.length || 0) * (SCENARIO_HEIGHT + 5) + 10;
}

export function layoutBoard(board: LoadedBoard): CanvasObject[] {
  const objects: CanvasObject[] = [];
  const { manifest, slices } = board;

  // Collect unique actors and check which have images
  const actors = new Set<string>();
  const actorHasImage = new Map<string, boolean>();
  for (const slice of slices.values()) {
    actors.add(slice.actor);
    if (slice.image) {
      actorHasImage.set(slice.actor, true);
    }
  }
  const actorList = Array.from(actors);

  // Calculate max scenarios height
  let maxScenariosHeight = LANE_HEIGHT;
  for (const slice of slices.values()) {
    maxScenariosHeight = Math.max(maxScenariosHeight, calcScenariosHeight(slice));
  }

  // Calculate Y positions - slice names at top
  const sliceNameY = 20;
  const actorLaneY: Record<string, number> = {};
  const actorLaneHeight: Record<string, number> = {};
  let currentY = 80;
  for (const actor of actorList) {
    actorLaneY[actor] = currentY;
    const height = actorHasImage.get(actor) ? LANE_HEIGHT_WITH_IMAGE : LANE_HEIGHT;
    actorLaneHeight[actor] = height;
    currentY += height;
  }
  const commandY = currentY;
  currentY += LANE_HEIGHT;
  const eventY = currentY;
  currentY += LANE_HEIGHT;
  const scenarioY = currentY;

  // First pass: calculate column widths (slices AND stories)
  const columnWidths: number[] = [];
  const columnEntries: FlowEntry[] = [];
  for (const entry of manifest.flow) {
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

  // Calculate column X positions
  const columnX: number[] = [];
  let x = SWIMLANE_PADDING;
  for (const w of columnWidths) {
    columnX.push(x);
    x += w + COLUMN_PADDING;
  }
  const totalWidth = x + SWIMLANE_PADDING;

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
      addStory(objects, entry, colX, colW, colIndex, sliceNameY, commandY);
    } else {
      const slice = slices.get(entry.name);
      if (!slice) continue;

      const actorY = actorLaneY[slice.actor];
      const laneHeight = actorLaneHeight[slice.actor];
      if (slice.type === 'change') {
        addChangeSlice(objects, slice as ChangeSlice, colX, colW, colIndex, sliceNameY, actorY, laneHeight, commandY, eventY, scenarioY);
      } else {
        addViewSlice(objects, slice as ViewSlice, colX, colW, colIndex, sliceNameY, actorY, laneHeight, commandY, eventY, scenarioY);
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
  commandY: number
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

  // Story card in command lane
  objects.push({
    id: `story-${entry.name}`,
    type: 'story',
    x,
    y: commandY,
    width: colWidth,
    height: STORY_HEIGHT,
    label: `(${entry.sliceRef})`,
    color: COLORS.story,
    metadata: { sliceRef: entry.sliceRef, description: entry.description, instance: entry.instance },
    sliceIndex: colIndex,
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
  commandY: number,
  eventY: number,
  scenarioY: number
): void {
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

  // Trigger: endpoint or external event (at bottom of actor lane)
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
  } else {
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
  scenarioY: number
): void {
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
