(function () {
    const selector = '[data-e2e="room-chat-like-btn"]';
    const runForMs = 100 * 60 * 1000; // 10 minutes
    const minDelay = 500;
    const maxDelay = 1500;

    let stopped = false;
    let timeoutId = null;
    const startedAt = Date.now();

    function getRandomDelay() {
        return Math.floor(Math.random() * (maxDelay - minDelay + 1)) + minDelay;
    }

    function clickLoop() {
        if (stopped) return;

        if (Date.now() - startedAt >= runForMs) {
            stopClicking();
            console.log("Auto-stopped after 10 minutes.");
            return;
        }

        const el = document.querySelector(selector);

        if (!el) {
            console.log("Element not found:", selector);
            stopClicking();
            return;
        }

        el.click();

        timeoutId = setTimeout(clickLoop, getRandomDelay());
    }

    function stopClicking() {
        stopped = true;

        if (timeoutId) {
            clearTimeout(timeoutId);
            timeoutId = null;
        }

        console.log("Stopped.");
    }

    window.stopClicking = stopClicking;

    clickLoop();
})();