import './style.css';
import {render} from 'preact';
import {App} from './App.js';
import {html} from './html.js';

render(html`<${App} />`, document.body);
