// Board manifest structure
export interface BoardManifest {
  name: string;
  actors: string[];
  contexts: ContextEntry[];
  flow: FlowEntry[];
  errors?: string[];
}

export interface ContextEntry {
  name: string;
  description?: string;
  chapters: ChapterEntry[];
}

export interface ChapterEntry {
  name: string;
  description?: string;
  flowIndices: number[];
}

export interface FlowEntry {
  index: number;
  kind: 'slice' | 'story';
  type?: 'change' | 'view' | 'automation';
  name: string;
  file?: string;
  sliceRef?: string;
  description?: string;
  instance?: Record<string, unknown>;
  emits?: StoryEventInstance[];
  image?: string;
}

export interface StoryEventInstance {
  type: string;
  values?: Record<string, unknown>;
}

// Slice data structures
export interface ChangeSlice {
  kind: 'slice';
  type: 'change';
  name: string;
  actor: string;
  image?: string;
  devstatus?: string;
  trigger: Trigger;
  command: Command;
  emits: EventEmit[];
  scenarios: ChangeScenario[];
}

export interface ViewSlice {
  kind: 'slice';
  type: 'view';
  name: string;
  actor: string;
  image?: string;
  devstatus?: string;
  endpoint: Endpoint;
  query: QueryItem[];
  readModel: ReadModel;
  scenarios: ViewScenario[];
}

// AutomationSlice - event-triggered automation (no actor)
export interface AutomationSlice {
  kind: 'slice';
  type: 'automation';
  name: string;
  image?: string;
  devstatus?: string;
  trigger: ExternalEventTrigger | InternalEventTrigger;
  command: Command;
  emits: EventEmit[];
  scenarios: ChangeScenario[];
}

export type Slice = ChangeSlice | ViewSlice | AutomationSlice;

export interface Endpoint {
  verb: string;
  path: string;
  params?: Record<string, string>;
  body?: Record<string, string>;
}

export interface ExternalEvent {
  name: string;
  source: string;
  fields: Record<string, string>;
}

export interface EndpointTrigger {
  kind: 'endpoint';
  endpoint: Endpoint;
}

export interface ExternalEventTrigger {
  kind: 'externalEvent';
  externalEvent: ExternalEvent;
}

export interface InternalEvent {
  eventType: string;
  fields: Record<string, string>;
}

export interface InternalEventTrigger {
  kind: 'internalEvent';
  internalEvent: InternalEvent;
}

export type Trigger = EndpointTrigger | ExternalEventTrigger | InternalEventTrigger;

export interface Command {
  fields: Record<string, string>;
  query: QueryItem[];
}

export interface QueryItem {
  types: string[];
  tags: TagBinding[];
}

export interface TagBinding {
  tag: string;
  param: string;
}

export interface EventEmit {
  type: string;
  fields: Record<string, string>;
  tags: string[];
  mapping?: Record<string, string>;
}

export interface ReadModel {
  name: string;
  cardinality: 'single' | 'multiple';
  fields: Record<string, unknown>;
  mapping: Record<string, string>;
}

export interface ChangeScenario {
  name: string;
  given: (string | { type: string; values: Record<string, unknown> })[];
  when: { command: string; values?: Record<string, unknown> };
  then: { success: boolean; events?: string[]; error?: string };
}

export interface ViewScenario {
  name: string;
  given: (string | { type: string; values: Record<string, unknown> })[];
  query: Record<string, unknown>;
  expect: Record<string, unknown>;
}

// Canvas object types
export type ObjectType = 'command' | 'event' | 'read-model' | 'endpoint' | 'external-event' | 'watcher' | 'swimlane' | 'slice-name' | 'slice-border' | 'scenario' | 'mockup' | 'story' | 'chapter-lane';

export interface CanvasObject {
  id: string;
  type: ObjectType;
  x: number;
  y: number;
  width: number;
  height: number;
  label: string;
  color: string;
  metadata?: Record<string, unknown>;
  sliceIndex?: number;
}

export interface Viewport {
  x: number;
  y: number;
  zoom: number;
  width: number;
  height: number;
}

export interface BoundingBox {
  minX: number;
  minY: number;
  maxX: number;
  maxY: number;
}
