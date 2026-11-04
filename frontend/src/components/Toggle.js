import {html} from '../html.js';

// A switch that is a button, so the D-pad can focus it and A can flip it.
export function Toggle({label, checked, onChange}) {
    return html`
        <button class="switch" type="button" role="switch" aria-checked=${checked}
                onClick=${() => onChange(!checked)}>
            <span class="switch-track"><span class="switch-thumb"></span></span>
            <span class="switch-label">${label}</span>
        </button>`;
}
