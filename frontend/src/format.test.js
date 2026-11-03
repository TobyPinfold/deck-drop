import test from 'node:test';
import assert from 'node:assert/strict';
import {appendDigit, errorMessage, formatBytes, transferDetailText, transferPercent, transferStatusText, secondsLeft} from './format.js';

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

test('appendDigit stops at the maximum length', () => {
    assert.equal(appendDigit('12345', '6', 6), '123456');
    assert.equal(appendDigit('123456', '7', 6), '123456');
});

test('secondsLeft counts down and never goes negative', () => {
    const expires = new Date(10_000).toISOString();
    assert.equal(secondsLeft(expires, 7_500), 3);
    assert.equal(secondsLeft(expires, 20_000), 0);
});
