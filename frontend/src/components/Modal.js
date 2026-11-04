import {useEffect, useRef} from 'preact/hooks';
import {html} from '../html.js';

// A dialog over the page. Focus moves into it when it opens and back to what
// opened it when it closes. Navigation stays inside while it is open (see
// input.js), and B presses the close button.
export function Modal({title, onClose, closeLabel = 'Close', focusKey, children, role = 'dialog'}) {
    const dialog = useRef(null);

    useEffect(() => {
        const opener = document.activeElement;
        // The page behind is inert until this dialog has gone, so wait for that.
        return () => requestAnimationFrame(() => {
            if (opener?.isConnected) {
                opener.focus();
            }
        });
    }, []);

    // Focus moves to the first control when the dialog opens, and again when
    // focusKey changes, because the control that had focus is usually gone then.
    useEffect(() => {
        const first = dialog.current.querySelector('[data-autofocus]:not([disabled])')
            ?? dialog.current.querySelector('button:not([disabled]), input:not([disabled])');
        first?.focus();
    }, [focusKey]);

    return html`
        <div class="modal-backdrop">
            <section class="modal" role=${role} aria-modal="true" aria-label=${title} ref=${dialog}>
                <h2>${title}</h2>
                ${children}
                <div class="modal-footer">
                    <button class="action" type="button" data-modal-close onClick=${onClose}>${closeLabel}</button>
                </div>
            </section>
        </div>`;
}
