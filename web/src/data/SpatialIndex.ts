import RBush from 'rbush';
import type { CanvasObject, BoundingBox, Viewport } from './types';

interface IndexedObject extends BoundingBox {
  id: string;
}

export class SpatialIndex {
  private tree = new RBush<IndexedObject>();
  private objects = new Map<string, CanvasObject>();

  load(objects: CanvasObject[]): void {
    this.objects.clear();
    const items: IndexedObject[] = [];

    for (const obj of objects) {
      this.objects.set(obj.id, obj);
      items.push({
        minX: obj.x,
        minY: obj.y,
        maxX: obj.x + obj.width,
        maxY: obj.y + obj.height,
        id: obj.id,
      });
    }

    this.tree.clear();
    this.tree.load(items);
  }

  getVisible(viewport: Viewport): CanvasObject[] {
    const bounds = this.viewportToWorld(viewport);
    const hits = this.tree.search(bounds);
    return hits.map(h => this.objects.get(h.id)!).filter(Boolean);
  }

  hitTest(worldX: number, worldY: number): CanvasObject | null {
    const hits = this.tree.search({
      minX: worldX,
      minY: worldY,
      maxX: worldX,
      maxY: worldY,
    });

    // Prefer interactive objects over background (swimlanes, slice-names)
    const background = new Set(['swimlane', 'slice-name']);
    let fallback: CanvasObject | null = null;

    for (const hit of hits) {
      const obj = this.objects.get(hit.id);
      if (obj && this.containsPoint(obj, worldX, worldY)) {
        if (!background.has(obj.type)) {
          return obj;
        }
        fallback = obj;
      }
    }
    return fallback;
  }

  private viewportToWorld(viewport: Viewport): BoundingBox {
    return {
      minX: viewport.x,
      minY: viewport.y,
      maxX: viewport.x + viewport.width / viewport.zoom,
      maxY: viewport.y + viewport.height / viewport.zoom,
    };
  }

  private containsPoint(obj: CanvasObject, x: number, y: number): boolean {
    return x >= obj.x && x <= obj.x + obj.width &&
           y >= obj.y && y <= obj.y + obj.height;
  }
}
