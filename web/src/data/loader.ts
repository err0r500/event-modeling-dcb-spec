import type { BoardManifest, Slice } from './types';

export const BOARD_PATH = '/.board';

// Translate CUE constraint errors to user-friendly messages
function translateError(error: string): string {
    // Type mismatch: "slice_X_field_Y_type: expected int, got string"
    const typeMatch = error.match(/\.slice_(\w+)_field_(\w+)_type:\s*expected\s+(\w+),\s*got\s+(\w+)/);
    if (typeMatch) {
        return `Slice '${typeMatch[1]}': command field '${typeMatch[2]}', endpoint says ${typeMatch[3]}, got ${typeMatch[4]}`;
    }

    // Emit field type mismatch: "slice_X_emit_Y_field_Z_type: expected..."
    const emitTypeMatch = error.match(/\.slice_(\w+)_emit_(\w+)_field_(\w+)_type:\s*expected\s+(\w+),\s*got\s+(\w+)/);
    if (emitTypeMatch) {
        return `Slice '${emitTypeMatch[1]}' emit '${emitTypeMatch[2]}': field '${emitTypeMatch[3]}' expected ${emitTypeMatch[4]}, got ${emitTypeMatch[5]}`;
    }

    // View mapping type: "view_X_mapping_Y_type: expected..."
    const viewTypeMatch = error.match(/\.view_(\w+)_mapping_(\w+)_type:\s*expected\s+(\w+),\s*got\s+(\w+)/);
    if (viewTypeMatch) {
        return `View '${viewTypeMatch[1]}': mapping '${viewTypeMatch[2]}' expected ${viewTypeMatch[3]}, got ${viewTypeMatch[4]}`;
    }

    // Tag value type mismatch: "tags.0.value: conflicting values X and Y (mismatched types..."
    const tagValueMatch = error.match(/tags\.\d+\.value:\s*conflicting values (\S+) and (\S+)/);
    if (tagValueMatch) {
        console.log(error)
        return `Tag value type mismatch: expected ${tagValueMatch[1]}, got ${tagValueMatch[2]}`;
    }

    // Extract constraint name from error (e.g., "cartBoard.slice_AddItem_scenario0_given_CartCleared_must_be_in_query: conflicting...")
    const match = error.match(/\.([a-zA-Z0-9_]+):\s*conflicting/);
    if (!match) return error;

    const constraint = match[1];

    // slice_<name>_scenario<n>_given_<event>_must_be_in_query
    const givenMatch = constraint.match(/^slice_(\w+)_scenario(\d+)_given_(\w+)_must_be_in_query$/);
    if (givenMatch) {
        return `Slice '${givenMatch[1]}' scenario #${givenMatch[2]}: given event '${givenMatch[3]}' not in command.query`;
    }

    // slice_<name>_scenario<n>_then_<event>_must_be_in_emits
    const thenMatch = constraint.match(/^slice_(\w+)_scenario(\d+)_then_(\w+)_must_be_in_emits$/);
    if (thenMatch) {
        return `Slice '${thenMatch[1]}' scenario #${thenMatch[2]}: then event '${thenMatch[3]}' not in emits`;
    }

    // slice_<name>_scenario<n>_command_must_match
    const cmdMatch = constraint.match(/^slice_(\w+)_scenario(\d+)_command_must_match$/);
    if (cmdMatch) {
        return `Slice '${cmdMatch[1]}' scenario #${cmdMatch[2]}: command name mismatch`;
    }

    // slice_<name>_field_<field>_must_come_from_trigger
    const fieldMatch = constraint.match(/^slice_(\w+)_field_(\w+)_must_come_from_trigger$/);
    if (fieldMatch) {
        return `Slice '${fieldMatch[1]}': field '${fieldMatch[2]}' not in trigger`;
    }

    // automation_<name>_field_<field>_must_come_from_trigger
    const autoFieldMatch = constraint.match(/^automation_(\w+)_field_(\w+)_must_come_from_trigger$/);
    if (autoFieldMatch) {
        return `Automation '${autoFieldMatch[1]}': field '${autoFieldMatch[2]}' not in trigger`;
    }

    // automation_<name>_internalEvent_<event>_must_be_emitted_before
    const autoEventMatch = constraint.match(/^automation_(\w+)_internalEvent_(\w+)_must_be_emitted_before$/);
    if (autoEventMatch) {
        return `Automation '${autoEventMatch[1]}': trigger event '${autoEventMatch[2]}' not emitted before`;
    }

    // slice_<name>_event_<event>_must_be_emitted_before
    const emitBeforeMatch = constraint.match(/^slice_(\w+)_event_(\w+)_must_be_emitted_before$/);
    if (emitBeforeMatch) {
        return `Slice '${emitBeforeMatch[1]}': event '${emitBeforeMatch[2]}' not emitted before this view`;
    }

    // slice_<name>_event_<event>_must_have_tag_<tag>
    const tagMatch = constraint.match(/^slice_(\w+)_event_(\w+)_must_have_tag_(\w+)$/);
    if (tagMatch) {
        return `Slice '${tagMatch[1]}': event '${tagMatch[2]}' missing tag '${tagMatch[3]}'`;
    }

    // view_<name>_field_<field>_must_come_from_events_or_computed
    const viewFieldMatch = constraint.match(/^view_(\w+)_field_(\w+)_must_come_from_events_or_computed$/);
    if (viewFieldMatch) {
        return `View '${viewFieldMatch[1]}': field '${viewFieldMatch[2]}' not from events/computed`;
    }

    // Fallback: just show constraint name
    return constraint.replace(/_/g, ' ');
}

export interface LoadedBoard {
    manifest: BoardManifest;
    slices: Map<string, Slice>;
    error?: string;
}

export async function loadBoard(): Promise<LoadedBoard> {
    const manifestRes = await fetch(`${BOARD_PATH}/board.json`);
    if (!manifestRes.ok) {
        throw new Error(`Failed to load board.json: ${manifestRes.status}`);
    }
    const data = await manifestRes.json();

    // Check for error response (errors array or null flow)
    if (data.errors?.length || !data.flow) {
        const translatedErrors = (data.errors || ['Invalid board data']).map(translateError);
        return {
            manifest: { name: 'Error', actors: [], contexts: [], flow: [] },
            slices: new Map(),
            error: translatedErrors.join('\n'),
        };
    }

    const manifest: BoardManifest = data;
    const slices = new Map<string, Slice>();

    // Load slice files in parallel, then populate Map in flow order (for deterministic actor lane ordering)
    const sliceEntries = manifest.flow.filter(entry => entry.kind === 'slice' && entry.file);
    const fetchResults = await Promise.all(
        sliceEntries.map(async entry => {
            const res = await fetch(`${BOARD_PATH}/${entry.file}`);
            if (res.ok) {
                return { name: entry.name, slice: await res.json() as Slice };
            }
            return null;
        })
    );
    // Insert in flow order
    for (const result of fetchResults) {
        if (result) slices.set(result.name, result.slice);
    }

    return { manifest, slices };
}

async function hashText(text: string): Promise<string> {
    // crypto.subtle not available in insecure contexts (Safari)
    if (!crypto.subtle) {
        // Simple hash fallback - just use length + first/last chars
        return `${text.length}-${text.slice(0, 100)}-${text.slice(-100)}`;
    }
    const encoder = new TextEncoder();
    const data = encoder.encode(text);
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

export function watchBoard(callback: () => void): void {
    let lastHash = '';

    setInterval(async () => {
        try {
            // Fetch manifest
            const manifestRes = await fetch(`${BOARD_PATH}/board.json`, { cache: 'no-store' });
            const manifestText = await manifestRes.text();
            const manifest = JSON.parse(manifestText);

            // Fetch all slice files
            const sliceFiles = (manifest.flow || [])
                .filter((e: any) => e.file)
                .map((e: any) => e.file);

            const sliceTexts = await Promise.all(
                sliceFiles.map(async (file: string) => {
                    const res = await fetch(`${BOARD_PATH}/${file}`, { cache: 'no-store' });
                    return res.ok ? res.text() : '';
                })
            );

            // Hash everything together
            const allContent = manifestText + sliceTexts.join('');
            const hash = await hashText(allContent);

            if (lastHash && hash !== lastHash) {
                callback();
            }
            lastHash = hash;
        } catch {
            // ignore
        }
    }, 500);
}
