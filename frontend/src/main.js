import './style.css';
import {Info} from '../bindings/github.com/TobyPinfold/deck-drop/service.js';

const status = document.querySelector('#status');

Info().then((info) => {
    status.textContent = info.running
        ? `Receiving on port ${info.port}`
        : `Backend not running: ${info.error}`;
});
