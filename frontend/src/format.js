// Pure display helpers shared by the components.

export function formatBytes(bytes) {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let value = bytes;
    let unit = 0;
    while (value >= 1000 && unit < units.length - 1) {
        value /= 1000;
        unit++;
    }
    return `${unit === 0 ? value : value.toFixed(1)} ${units[unit]}`;
}

export function transferPercent(transfer) {
    if (transfer.status === 'done') {
        return 100;
    }
    return transfer.total > 0 ? (transfer.sent / transfer.total) * 100 : 0;
}

export function transferStatusText(transfer) {
    if (transfer.status === 'failed') {
        return `Failed: ${transfer.error}`;
    }
    if (transfer.status === 'interrupted') {
        return 'Interrupted, waiting for the other device to continue';
    }
    if (transfer.status === 'waiting') {
        return 'Waiting for the other device to accept';
    }
    return transfer.status;
}

export function transferDetailText(transfer) {
    if (transfer.total <= 0) {
        return '';
    }
    const progress = `${formatBytes(transfer.sent)} of ${formatBytes(transfer.total)}`;
    const showSpeed = (transfer.status === 'sending' || transfer.status === 'receiving') && transfer.speed > 0;
    return showSpeed ? `${progress} · ${formatBytes(transfer.speed)}/s` : progress;
}

// How many files are currently coming in.
export function activeDownloads(transfers) {
    return transfers.filter((transfer) => transfer.direction === 'receive' && transfer.status === 'receiving').length;
}

export function errorMessage(error) {
    return error?.message ?? String(error);
}

// Adds digit to a code being typed, ignoring anything past maxLength.
export function appendDigit(code, digit, maxLength) {
    return code.length < maxLength ? code + digit : code;
}

// Whole seconds until expiresAt (an ISO time string), never below zero.
export function secondsLeft(expiresAt, nowMs) {
    return Math.max(0, Math.ceil((new Date(expiresAt).getTime() - nowMs) / 1000));
}

// "3 files (4.2 MB)" or, for one file, its name.
export function offerSummary(offer) {
    const size = formatBytes(offer.bytes);
    return offer.count === 1 ? `${offer.names[0]} (${size})` : `${offer.count} files (${size})`;
}
