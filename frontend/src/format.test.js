import test from 'node:test';
import assert from 'node:assert/strict';
import {errorMessage, formatBytes, transferDetailText, transferPercent, transferStatusText} from './format.js';

test('formatBytes uses decimal units', () => {
    assert.equal(formatBytes(999), '999 B');
    assert.equal(formatBytes(1500), '1.5 KB');
    assert.equal(formatBytes(2_500_000_000), '2.5 GB');
});

test('transferPercent is full when done and safe when the total is unknown', () => {
    assert.equal(transferPercent({status: 'done', sent: 0, total: 0}), 100);
    assert.equal(transferPercent({status: 'sending', sent: 1, total: 4}), 25);
    assert.equal(transferPercent({status: 'queued', sent: 0, total: 0}), 0);
});

test('transferStatusText shows the error for failed transfers', () => {
    assert.equal(transferStatusText({status: 'failed', error: 'boom'}), 'Failed: boom');
    assert.equal(transferStatusText({status: 'paused'}), 'paused');
});

test('transferDetailText shows speed only while sending', () => {
    const transfer = {status: 'sending', sent: 1000, total: 2000, speed: 500};
    assert.equal(transferDetailText(transfer), '1.0 KB of 2.0 KB · 500 B/s');
    assert.equal(transferDetailText({...transfer, status: 'paused'}), '1.0 KB of 2.0 KB');
    assert.equal(transferDetailText({...transfer, total: 0}), '');
});

test('errorMessage accepts errors and plain values', () => {
    assert.equal(errorMessage(new Error('nope')), 'nope');
    assert.equal(errorMessage('plain'), 'plain');
});
