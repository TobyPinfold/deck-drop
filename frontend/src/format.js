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
    return transfer.status === 'failed' ? `Failed: ${transfer.error}` : transfer.status;
}

export function transferDetailText(transfer) {
    if (transfer.total <= 0) {
        return '';
    }
    const progress = `${formatBytes(transfer.sent)} of ${formatBytes(transfer.total)}`;
    const showSpeed = transfer.status === 'sending' && transfer.speed > 0;
    return showSpeed ? `${progress} · ${formatBytes(transfer.speed)}/s` : progress;
}

export function errorMessage(error) {
    return error?.message ?? String(error);
}
