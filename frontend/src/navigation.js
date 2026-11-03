// Spatial navigation: picks which element to focus when moving in a direction.

// Elements that do not overlap the current one across the direction of travel
// only win when nothing is lined up with it.
const MISALIGNED_PENALTY = 1e6;

const centre = (rect) => ({x: (rect.left + rect.right) / 2, y: (rect.top + rect.bottom) / 2});

function gap(startA, endA, startB, endB) {
    return Math.max(0, startB - endA, startA - endB);
}

// findNext returns the index of the candidate rectangle to move to from
// current in direction ('up', 'down', 'left' or 'right'), or -1 if there is none.
export function findNext(current, candidates, direction) {
    const from = centre(current);
    let best = -1;
    let bestScore = Infinity;

    candidates.forEach((candidate, index) => {
        const to = centre(candidate);
        const horizontal = direction === 'left' || direction === 'right';
        const sign = direction === 'right' || direction === 'down' ? 1 : -1;
        const primary = (horizontal ? to.x - from.x : to.y - from.y) * sign;
        if (primary <= 0) {
            return;
        }

        const offAxis = horizontal
            ? gap(current.top, current.bottom, candidate.top, candidate.bottom)
            : gap(current.left, current.right, candidate.left, candidate.right);
        const score = primary + 2 * offAxis + (offAxis > 0 ? MISALIGNED_PENALTY : 0);
        if (score < bestScore) {
            best = index;
            bestScore = score;
        }
    });
    return best;
}
