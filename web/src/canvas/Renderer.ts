import type { CanvasObject, Viewport } from '../data/types';
import { SpatialIndex } from '../data/SpatialIndex';
import { BOARD_PATH } from '../data/loader';

export class Renderer {
    private ctx: CanvasRenderingContext2D;
    private index: SpatialIndex;
    private viewport: Viewport;
    private isDirty = true;
    private hoveredId: string | null = null;
    private highlightSet: Set<string> | null = null;
    private rafId: number | null = null;
    private imageCache: Map<string, HTMLImageElement> = new Map();
    private loadingImages: Set<string> = new Set();
    private dpr = Math.max(2, window.devicePixelRatio || 1);

    constructor(
        canvas: HTMLCanvasElement,
        index: SpatialIndex,
        viewport: Viewport
    ) {
        this.ctx = canvas.getContext('2d')!;
        this.index = index;
        this.viewport = viewport;
    }

    setHovered(id: string | null): void {
        if (this.hoveredId !== id) {
            this.hoveredId = id;
            this.markDirty();
        }
    }

    setHighlightSet(ids: string[] | null): void {
        this.highlightSet = ids ? new Set(ids) : null;
        this.markDirty();
    }

    markDirty(): void {
        this.isDirty = true;
    }

    setDPR(dpr: number): void {
        this.dpr = dpr;
    }

    start(): void {
        this.render();
    }

    stop(): void {
        if (this.rafId) {
            cancelAnimationFrame(this.rafId);
        }
    }

    private loadImage(src: string): HTMLImageElement | null {
        if (this.imageCache.has(src)) {
            return this.imageCache.get(src)!;
        }
        if (this.loadingImages.has(src)) {
            return null;
        }
        this.loadingImages.add(src);
        const img = new Image();
        img.onload = () => {
            this.imageCache.set(src, img);
            this.loadingImages.delete(src);
            this.markDirty();
        };
        img.onerror = () => {
            this.loadingImages.delete(src);
        };
        img.src = src;
        return null;
    }

    private render(): void {
        if (this.isDirty) {
            this.draw();
            this.isDirty = false;
        }
        this.rafId = requestAnimationFrame(() => this.render());
    }

    private draw(): void {
        const ctx = this.ctx;
        const { x, y, zoom, width, height } = this.viewport;

        ctx.setTransform(1, 0, 0, 1, 0, 0);
        ctx.clearRect(0, 0, width * this.dpr, height * this.dpr);

        // Background
        ctx.fillStyle = '#1e1e2e';
        ctx.fillRect(0, 0, width * this.dpr, height * this.dpr);

        ctx.save();
        ctx.scale(this.dpr, this.dpr);
        ctx.scale(zoom, zoom);
        ctx.translate(-x, -y);

        // Draw grid
        this.drawGrid();

        // Get visible objects sorted by type for layering
        const visible = this.index.getVisible(this.viewport);
        const swimlanes = visible.filter(o => o.type === 'swimlane');
        const chapterLanes = visible.filter(o => o.type === 'chapter-lane');
        const sliceBorders = visible.filter(o => o.type === 'slice-border');
        const mockups = visible.filter(o => o.type === 'mockup');
        const sliceNames = visible.filter(o => o.type === 'slice-name');
        const others = visible.filter(o => o.type !== 'swimlane' && o.type !== 'chapter-lane' && o.type !== 'slice-border' && o.type !== 'slice-name' && o.type !== 'mockup');

        // Swimlanes first (background)
        for (const obj of swimlanes) {
            this.drawSwimlane(obj);
        }

        // Chapter lanes
        for (const obj of chapterLanes) {
            this.drawChapterLane(obj);
        }

        // Slice borders (behind content)
        for (const obj of sliceBorders) {
            this.drawSliceBorder(obj);
        }

        // Mockups
        for (const obj of mockups) {
            this.drawMockup(obj);
        }

        // Slice names
        for (const obj of sliceNames) {
            this.drawSliceName(obj);
        }

        // Then other objects
        for (const obj of others) {
            if (obj.type === 'scenario') {
                this.drawScenario(obj);
            } else if (obj.type === 'story') {
                this.drawStory(obj);
            } else {
                this.drawObject(obj);
            }
        }

        // Instance panels on top (z-index)
        const stories = others.filter(o => o.type === 'story');
        for (const obj of stories) {
            const instance = obj.metadata?.instance as Record<string, unknown> | undefined;
            if (instance && Object.keys(instance).length > 0) {
                this.drawInstancePanel(obj, instance);
            }
        }

        ctx.restore();
    }

    private drawGrid(): void {
        const ctx = this.ctx;
        const { x, y, zoom, width, height } = this.viewport;

        const gridSize = 50;
        const startX = Math.floor(x / gridSize) * gridSize;
        const startY = Math.floor(y / gridSize) * gridSize;
        const endX = x + width / zoom;
        const endY = y + height / zoom;

        ctx.strokeStyle = '#313244';
        ctx.lineWidth = 1 / zoom;
        ctx.beginPath();

        for (let gx = startX; gx <= endX; gx += gridSize) {
            ctx.moveTo(gx, startY);
            ctx.lineTo(gx, endY);
        }
        for (let gy = startY; gy <= endY; gy += gridSize) {
            ctx.moveTo(startX, gy);
            ctx.lineTo(endX, gy);
        }

        ctx.stroke();
    }

    private drawSwimlane(obj: CanvasObject): void {
        const ctx = this.ctx;

        ctx.fillStyle = obj.color;
        ctx.fillRect(obj.x, obj.y, obj.width, obj.height);

        // Label on left
        if (this.viewport.zoom > 0.2) {
            const zoomFactor = Math.pow(this.viewport.zoom, 0.5);
            ctx.fillStyle = '#6c7086';
            ctx.font = `${12 / zoomFactor}px system-ui`;
            ctx.textAlign = 'left';
            ctx.textBaseline = 'middle';
            ctx.fillText(obj.label, obj.x + 10, obj.y + obj.height / 2);
        }
    }

    private drawChapterLane(obj: CanvasObject): void {
        const ctx = this.ctx;

        // Fill background
        ctx.fillStyle = obj.color;
        this.roundRect(obj.x, obj.y, obj.width, obj.height, 4);
        ctx.fill();

        // Border
        ctx.strokeStyle = '#585b70';
        ctx.lineWidth = 1 / this.viewport.zoom;
        this.roundRect(obj.x, obj.y, obj.width, obj.height, 4);
        ctx.stroke();

        // Label centered
        if (this.viewport.zoom > 0.15) {
            const zoomFactor = Math.pow(this.viewport.zoom, 0.5);
            ctx.fillStyle = '#89b4fa'; // blue color for chapters
            ctx.font = `600 ${14 / zoomFactor}px system-ui`;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(obj.label, obj.x + obj.width / 2, obj.y + obj.height / 2);
        }
    }

    private drawSliceBorder(obj: CanvasObject): void {
        const ctx = this.ctx;

        // Transparent fill with rounded border
        ctx.strokeStyle = obj.color;
        ctx.lineWidth = 1.5 / this.viewport.zoom;
        this.roundRect(obj.x, obj.y, obj.width, obj.height, 8);
        ctx.stroke();
    }

    private drawMockup(obj: CanvasObject): void {
        const ctx = this.ctx;
        const src = obj.metadata?.src as string;
        if (!src) return;

        const fullSrc = `${BOARD_PATH}/${src}`;
        const img = this.loadImage(fullSrc);

        // Draw rounded border
        ctx.fillStyle = obj.color;
        ctx.strokeStyle = '#45475a';
        ctx.lineWidth = 1 / this.viewport.zoom;
        this.roundRect(obj.x, obj.y, obj.width, obj.height, 6);
        ctx.fill();
        ctx.stroke();

        if (img) {
            // Draw image scaled to fit, preserving aspect ratio
            const padding = 4;
            const availW = obj.width - padding * 2;
            const availH = obj.height - padding * 2;
            const scale = Math.min(availW / img.width, availH / img.height);
            const drawW = img.width * scale;
            const drawH = img.height * scale;
            const drawX = obj.x + (obj.width - drawW) / 2;
            const drawY = obj.y + (obj.height - drawH) / 2;

            ctx.save();
            // Clip to rounded rect
            this.roundRect(obj.x + padding, obj.y + padding, availW, availH, 4);
            ctx.clip();
            ctx.drawImage(img, drawX, drawY, drawW, drawH);
            ctx.restore();
        } else {
            // Show loading placeholder
            if (this.viewport.zoom > 0.2) {
                ctx.fillStyle = '#6c7086';
                const zoomFactor = Math.pow(this.viewport.zoom, 0.5);
                ctx.font = `${10 / zoomFactor}px system-ui`;
                ctx.textAlign = 'center';
                ctx.textBaseline = 'middle';
                ctx.fillText('Loading...', obj.x + obj.width / 2, obj.y + obj.height / 2);
            }
        }
    }

    private drawSliceName(obj: CanvasObject): void {
        const ctx = this.ctx;

        if (this.viewport.zoom > 0.15) {
            const zoomFactor = Math.pow(this.viewport.zoom, 0.5);
            const devstatus = obj.metadata?.devstatus as string | undefined;

            // Draw slice name (shift up if devstatus present)
            const titleY = devstatus ? obj.y + obj.height / 2 - 8 : obj.y + obj.height / 2;
            ctx.fillStyle = obj.color;
            ctx.font = `600 ${16 / zoomFactor}px system-ui`;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(obj.label, obj.x + obj.width / 2, titleY);

            // Draw devstatus below title
            if (devstatus) {
                const statusColors: Record<string, string> = {
                    'specifying': '#6c7086',
                    'todo': '#89b4fa',
                    'doing': '#f9e2af',
                    'done': '#a6e3a1',
                };
                const statusColor = statusColors[devstatus] || '#6c7086';
                ctx.fillStyle = statusColor;
                ctx.font = `500 ${11 / zoomFactor}px system-ui`;
                ctx.fillText(devstatus, obj.x + obj.width / 2, obj.y + obj.height / 2 + 10);
            }
        }
    }

    private drawStory(obj: CanvasObject): void {
        const ctx = this.ctx;
        const isHovered = obj.id === this.hoveredId;
        const isDimmed = this.highlightSet !== null && !this.highlightSet.has(obj.id);

        if (isDimmed) {
            ctx.globalAlpha = 0.2;
        }

        if (isHovered) {
            ctx.shadowColor = 'rgba(0, 0, 0, 0.3)';
            ctx.shadowBlur = 10 / this.viewport.zoom;
            ctx.shadowOffsetY = 4 / this.viewport.zoom;
        }

        // Rounded rectangle (no border)
        ctx.fillStyle = obj.color;
        this.roundRect(obj.x, obj.y, obj.width, obj.height, 6);
        ctx.fill();

        ctx.shadowColor = 'transparent';

        // Label - reference name
        if (this.viewport.zoom > 0.15) {
            ctx.fillStyle = '#1e1e2e';
            const fontSize = Math.min(13, obj.width / 8);
            const zoomFactor = Math.pow(this.viewport.zoom, 0.5);
            ctx.font = `400 ${fontSize / zoomFactor}px system-ui`;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';

            const desc = obj.metadata?.description as string | undefined;
            const labelY = desc ? obj.y + obj.height / 2 - 8 : obj.y + obj.height / 2;
            const text = this.truncateText(obj.label, obj.width - 16);
            ctx.fillText(text, obj.x + obj.width / 2, labelY);

            // Description below
            if (desc) {
                ctx.font = `400 ${10 / zoomFactor}px system-ui`;
                ctx.fillStyle = '#585b70';
                const truncatedSub = this.truncateText(desc, obj.width - 12);
                ctx.fillText(truncatedSub, obj.x + obj.width / 2, obj.y + obj.height / 2 + 10);
            }
        }

        if (isDimmed) {
            ctx.globalAlpha = 1.0;
        }
    }

    private drawInstancePanel(obj: CanvasObject, instance: Record<string, unknown>): void {
        const ctx = this.ctx;
        const isDimmed = this.highlightSet !== null && !this.highlightSet.has(obj.id);

        if (isDimmed) {
            ctx.globalAlpha = 0.2;
        }

        const zoomFactor = Math.max(1, Math.pow(this.viewport.zoom, 0.5));
        const baseFontSize = Math.min(13, obj.width / 10);
        const lineHeight = 18 / zoomFactor;
        const padding = 10 / zoomFactor;
        const gap = 6 / this.viewport.zoom;

        // Format instance lines
        const lines = Object.entries(instance).map(([k, v]) => {
            const valStr = JSON.stringify(v, null, 2);
            return `${k}: ${valStr}`;
        });

        // Flatten multiline JSON
        const flatLines: string[] = [];
        for (const line of lines) {
            flatLines.push(...line.split('\n'));
        }

        // Measure panel size
        ctx.font = `400 ${baseFontSize / zoomFactor}px monospace`;
        let maxWidth = 0;
        for (const line of flatLines) {
            maxWidth = Math.max(maxWidth, ctx.measureText(line).width);
        }
        const panelW = obj.width;
        const panelH = flatLines.length * lineHeight + padding * 2;
        const panelX = obj.x;
        const panelY = obj.y + obj.height + gap;

        // Background - same style as story card
        ctx.fillStyle = '#1e1e2e';
        ctx.strokeStyle = '#585b70';
        ctx.lineWidth = 1 / this.viewport.zoom;
        ctx.setLineDash([4 / this.viewport.zoom, 3 / this.viewport.zoom]);
        this.roundRect(panelX, panelY, panelW, panelH, 4);
        ctx.fill();
        ctx.stroke();
        ctx.setLineDash([]);

        // Text
        ctx.fillStyle = '#a6adc8';
        ctx.textAlign = 'left';
        ctx.textBaseline = 'top';
        for (let i = 0; i < flatLines.length; i++) {
            ctx.fillText(flatLines[i], panelX + padding, panelY + padding + i * lineHeight);
        }

        if (isDimmed) {
            ctx.globalAlpha = 1.0;
        }
    }

    private drawScenario(obj: CanvasObject): void {
        const ctx = this.ctx;
        const isHovered = obj.id === this.hoveredId;
        const isDimmed = this.highlightSet !== null && !this.highlightSet.has(obj.id);
        const isSuccess = obj.metadata?.isSuccess as boolean;
        const borderColor = isSuccess ? '#a6e3a1' : '#f38ba8'; // green / red

        if (isDimmed) {
            ctx.globalAlpha = 0.2;
        }

        if (isHovered) {
            ctx.shadowColor = 'rgba(0, 0, 0, 0.3)';
            ctx.shadowBlur = 10 / this.viewport.zoom;
            ctx.shadowOffsetY = 4 / this.viewport.zoom;
        }

        // Main rounded rectangle
        ctx.fillStyle = obj.color;
        ctx.strokeStyle = isHovered ? '#f5e0dc' : '#45475a';
        ctx.lineWidth = (isHovered ? 2 : 1) / this.viewport.zoom;
        this.roundRect(obj.x, obj.y, obj.width, obj.height, 6);
        ctx.fill();
        ctx.stroke();

        ctx.shadowColor = 'transparent';

        // Left border (colored bar)
        const barWidth = 10 / this.viewport.zoom;
        ctx.fillStyle = borderColor;
        ctx.beginPath();
        const r = 6 / this.viewport.zoom;
        ctx.moveTo(obj.x + r, obj.y);
        ctx.lineTo(obj.x + barWidth, obj.y);
        ctx.lineTo(obj.x + barWidth, obj.y + obj.height);
        ctx.lineTo(obj.x + r, obj.y + obj.height);
        ctx.quadraticCurveTo(obj.x, obj.y + obj.height, obj.x, obj.y + obj.height - r);
        ctx.lineTo(obj.x, obj.y + r);
        ctx.quadraticCurveTo(obj.x, obj.y, obj.x + r, obj.y);
        ctx.closePath();
        ctx.fill();

        // Label
        if (this.viewport.zoom > 0.15) {
            ctx.fillStyle = '#1e1e2e';
            const fontSize = Math.min(14, obj.width / 8);
            const zoomFactor = Math.pow(this.viewport.zoom, 0.5);
            ctx.font = `400 ${fontSize / zoomFactor}px system-ui`;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            const text = this.truncateText(obj.label, obj.width - 16);
            ctx.fillText(text, obj.x + obj.width / 2, obj.y + obj.height / 2);
        }

        if (isDimmed) {
            ctx.globalAlpha = 1.0;
        }
    }

    private drawObject(obj: CanvasObject): void {
        const ctx = this.ctx;
        const isHovered = obj.id === this.hoveredId;
        const isDimmed = this.highlightSet !== null && !this.highlightSet.has(obj.id);

        if (isDimmed) {
            ctx.globalAlpha = 0.2;
        }

        // Shadow for hovered
        if (isHovered) {
            ctx.shadowColor = 'rgba(0, 0, 0, 0.3)';
            ctx.shadowBlur = 10 / this.viewport.zoom;
            ctx.shadowOffsetY = 4 / this.viewport.zoom;
        }

        // Rounded rectangle
        ctx.fillStyle = obj.color;
        ctx.strokeStyle = isHovered ? '#f5e0dc' : '#45475a';
        ctx.lineWidth = (isHovered ? 2 : 1) / this.viewport.zoom;

        this.roundRect(obj.x, obj.y, obj.width, obj.height, 6);
        ctx.fill();
        ctx.stroke();

        ctx.shadowColor = 'transparent';

        // Label - font shrinks with zoom (partial compensation)
        if (this.viewport.zoom > 0.15) {
            ctx.fillStyle = '#1e1e2e';
            const fontSize = Math.min(14, obj.width / 8);
            const zoomFactor = Math.pow(this.viewport.zoom, 0.5); // partial compensation
            ctx.font = `400 ${fontSize / zoomFactor}px system-ui`;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';

            // For events with tags, shift label up to make room for tags
            const hasTags = obj.type === 'event' && Array.isArray(obj.metadata?.tags) && obj.metadata.tags.length > 0;
            const labelY = hasTags ? obj.y + obj.height / 2 - 8 : obj.y + obj.height / 2;

            const text = this.truncateText(obj.label, obj.width - 16);
            ctx.fillText(text, obj.x + obj.width / 2, labelY);

            // Tags for events
            if (hasTags) {
                const tags = obj.metadata!.tags as string[];
                const tagStr = tags.join(', ');
                ctx.font = `400 ${10 / zoomFactor}px system-ui`;
                ctx.fillStyle = '#45475a';
                const truncatedTags = this.truncateText(tagStr, obj.width - 12);
                ctx.fillText(truncatedTags, obj.x + obj.width / 2, obj.y + obj.height / 2 + 12);
            }
        }

        if (isDimmed) {
            ctx.globalAlpha = 1.0;
        }
    }

    private roundRect(x: number, y: number, w: number, h: number, r: number): void {
        const ctx = this.ctx;
        r = r / this.viewport.zoom;
        ctx.beginPath();
        ctx.moveTo(x + r, y);
        ctx.lineTo(x + w - r, y);
        ctx.quadraticCurveTo(x + w, y, x + w, y + r);
        ctx.lineTo(x + w, y + h - r);
        ctx.quadraticCurveTo(x + w, y + h, x + w - r, y + h);
        ctx.lineTo(x + r, y + h);
        ctx.quadraticCurveTo(x, y + h, x, y + h - r);
        ctx.lineTo(x, y + r);
        ctx.quadraticCurveTo(x, y, x + r, y);
        ctx.closePath();
    }

    private truncateText(text: string, maxWidth: number): string {
        const ctx = this.ctx;
        if (ctx.measureText(text).width <= maxWidth) {
            return text;
        }
        let truncated = text;
        while (truncated.length > 0 && ctx.measureText(truncated + '...').width > maxWidth) {
            truncated = truncated.slice(0, -1);
        }
        return truncated + '...';
    }
}
