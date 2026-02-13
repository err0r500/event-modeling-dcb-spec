import type { Viewport } from '../data/types';

export class InputHandler {
    private canvas: HTMLCanvasElement;
    private viewport: Viewport;
    private onUpdate: () => void;
    private isPanning = false;
    private lastPos: { x: number; y: number } | null = null;

    constructor(
        canvas: HTMLCanvasElement,
        viewport: Viewport,
        onUpdate: () => void
    ) {
        this.canvas = canvas;
        this.viewport = viewport;
        this.onUpdate = onUpdate;

        this.setupEvents();
    }

    private setupEvents(): void {
        const canvas = this.canvas;

        // Pan
        canvas.addEventListener('mousedown', (e) => {
            if (e.button === 0) {
                this.isPanning = true;
                this.lastPos = { x: e.clientX, y: e.clientY };
                canvas.classList.add('panning');
            }
        });

        canvas.addEventListener('mousemove', (e) => {
            if (this.isPanning && this.lastPos) {
                const dx = (e.clientX - this.lastPos.x) / this.viewport.zoom;
                const dy = (e.clientY - this.lastPos.y) / this.viewport.zoom;

                this.viewport.x -= dx;
                this.viewport.y -= dy;

                this.lastPos = { x: e.clientX, y: e.clientY };
                this.onUpdate();
            }
        });

        const endPan = () => {
            this.isPanning = false;
            this.lastPos = null;
            canvas.classList.remove('panning');
        };

        canvas.addEventListener('mouseup', endPan);
        canvas.addEventListener('mouseleave', endPan);

        // Zoom
        canvas.addEventListener('wheel', (e) => {
            e.preventDefault();

            const rect = canvas.getBoundingClientRect();
            const mouseX = e.clientX - rect.left;
            const mouseY = e.clientY - rect.top;

            const worldX = this.viewport.x + mouseX / this.viewport.zoom;
            const worldY = this.viewport.y + mouseY / this.viewport.zoom;

            const zoomFactor = e.deltaY > 0 ? 0.95 : 1.02;
            this.viewport.zoom = Math.max(0.1, Math.min(5, this.viewport.zoom * zoomFactor));

            this.viewport.x = worldX - mouseX / this.viewport.zoom;
            this.viewport.y = worldY - mouseY / this.viewport.zoom;

            this.onUpdate();
        }, { passive: false });

        // Touch support
        let lastTouchDist = 0;

        canvas.addEventListener('touchstart', (e) => {
            if (e.touches.length === 1) {
                this.isPanning = true;
                this.lastPos = { x: e.touches[0].clientX, y: e.touches[0].clientY };
            } else if (e.touches.length === 2) {
                lastTouchDist = Math.hypot(
                    e.touches[0].clientX - e.touches[1].clientX,
                    e.touches[0].clientY - e.touches[1].clientY
                );
            }
        });

        canvas.addEventListener('touchmove', (e) => {
            e.preventDefault();

            if (e.touches.length === 1 && this.isPanning && this.lastPos) {
                const dx = (e.touches[0].clientX - this.lastPos.x) / this.viewport.zoom;
                const dy = (e.touches[0].clientY - this.lastPos.y) / this.viewport.zoom;

                this.viewport.x -= dx;
                this.viewport.y -= dy;

                this.lastPos = { x: e.touches[0].clientX, y: e.touches[0].clientY };
                this.onUpdate();
            } else if (e.touches.length === 2) {
                const dist = Math.hypot(
                    e.touches[0].clientX - e.touches[1].clientX,
                    e.touches[0].clientY - e.touches[1].clientY
                );

                if (lastTouchDist > 0) {
                    const scale = dist / lastTouchDist;
                    this.viewport.zoom = Math.max(0.1, Math.min(5, this.viewport.zoom * scale));
                    this.onUpdate();
                }
                lastTouchDist = dist;
            }
        }, { passive: false });

        canvas.addEventListener('touchend', () => {
            this.isPanning = false;
            this.lastPos = null;
            lastTouchDist = 0;
        });
    }
}
