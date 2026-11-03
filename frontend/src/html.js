import {h} from 'preact';
import htm from 'htm';

// Tagged-template JSX replacement that needs no build step. Interpolated
// values are always escaped, so text from disk or the network is never HTML.
// Unlike HTML, void elements such as <input> must be self-closed (<input />).
export const html = htm.bind(h);
