import './style.css';
import {Info} from '../wailsjs/go/app/App';

const status = document.querySelector('#status');

Info().then((info) => {
    status.textContent = info.running
        ? `Receiving on port ${info.port}`
        : `Backend not running: ${info.error}`;
});
