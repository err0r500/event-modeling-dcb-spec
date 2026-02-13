import type { CanvasObject, Viewport } from '../data/types';
import { SpatialIndex } from '../data/SpatialIndex';

export class HitTester {
  private canvas: HTMLCanvasElement;
  private index: SpatialIndex;
  private viewport: Viewport;
  private onHover: (obj: CanvasObject | null) => void;
  private onClick: ((obj: CanvasObject | null) => void) | null = null;
  private throttleTimeout: number | null = null;

  constructor(
    canvas: HTMLCanvasElement,
    index: SpatialIndex,
    viewport: Viewport,
    onHover: (obj: CanvasObject | null) => void,
    onClick?: (obj: CanvasObject | null) => void
  ) {
    this.canvas = canvas;
    this.index = index;
    this.viewport = viewport;
    this.onHover = onHover;
    this.onClick = onClick || null;

    this.setupEvents();
  }

  private screenToWorld(clientX: number, clientY: number): { x: number; y: number } {
    const rect = this.canvas.getBoundingClientRect();
    const mouseX = clientX - rect.left;
    const mouseY = clientY - rect.top;
    return {
      x: this.viewport.x + mouseX / this.viewport.zoom,
      y: this.viewport.y + mouseY / this.viewport.zoom,
    };
  }

  private setupEvents(): void {
    this.canvas.addEventListener('mousemove', (e) => {
      if (this.throttleTimeout) return;

      this.throttleTimeout = window.setTimeout(() => {
        this.throttleTimeout = null;
      }, 32);

      const { x: worldX, y: worldY } = this.screenToWorld(e.clientX, e.clientY);
      const hit = this.index.hitTest(worldX, worldY);

      // Skip swimlanes for hover
      if (hit && hit.type !== 'swimlane') {
        this.onHover(hit);
      } else {
        this.onHover(null);
      }
    });

    this.canvas.addEventListener('mouseleave', () => {
      this.onHover(null);
    });

    this.canvas.addEventListener('click', (e) => {
      if (!this.onClick) return;
      const { x: worldX, y: worldY } = this.screenToWorld(e.clientX, e.clientY);
      const hit = this.index.hitTest(worldX, worldY);
      if (hit && hit.type !== 'swimlane') {
        this.onClick(hit);
      }
    });
  }
}
