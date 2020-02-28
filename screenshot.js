const fs = require('fs');
const puppeteer = require('puppeteer');
hosts = {};
//now we read the host file
var hostFile = fs.readFileSync('hosts', 'utf8').split('\n');
var hosts = {};
for (var i = 0; i < hostFile.length; i++) {
    if (hostFile[i].charAt(0) == "#") {
        continue
    }
    var frags = hostFile[i].split(' ');
    if (frags.length > 1 && frags[0] === '0.0.0.0') {
        hosts[frags[1].trim()] = true;
    }
}
(async () => {
    const browser = await puppeteer.launch({ headless: true });
    const page = await browser.newPage();
    await page.setRequestInterception(true)
    page.on('request', request => {
        var domain = null;
        var frags = request.url().split('/');
        if (frags.length > 2) {
            domain = frags[2];
        }
        // just abort if found
        if (hosts[domain] === true) {
            request.abort();
        } else {
            request.continue();
        }
    });
    // Adjustments particular to this page to ensure we hit desktop breakpoint.
    page.setViewport({ width: 1000, height: 1000, deviceScaleFactor: 1 });
    await page.goto(process.argv[2]);
    await page.waitFor(5000);
    if (process.argv[4] == 'full') {
        await page.screenshot({
            path: process.argv[3],
            fullPage: true
        })
        await browser.close();
        return
    }
    /**
     * Takes a screenshot of a DOM element on the page, with optional padding.
     *
     * @param {!{path:string, selector:string, padding:(number|undefined)}=} opts
     * @return {!Promise<!Buffer>}
     */
    async function screenshotDOMElement(opts = {}) {
        const padding = 'padding' in opts ? opts.padding : 0;
        const path = 'path' in opts ? opts.path : null;
        const selector = opts.selector;
        if (!selector)
            throw Error('Please provide a selector.');
        const rect = await page.evaluate(selector => {
            const element = document.querySelector(selector);
            if (!element)
                return null;
            const { x, y, width, height } = element.getBoundingClientRect();
            return { left: x, top: y, width, height, id: element.id };
        }, selector);
        if (!rect)
            throw Error(`Could not find element that matches selector: ${selector}.`);
        return await page.screenshot({
            path,
            clip: {
                x: rect.left - padding,
                y: rect.top - padding,
                width: rect.width + padding * 2,
                height: rect.height + padding * 2
            }
        });
    }
    await screenshotDOMElement({
        path: process.argv[3],
        selector: process.argv[4],
        padding: 0
    });
    await browser.close();
})();
