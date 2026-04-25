import '@testing-library/jest-dom/vitest'
// Mantine requires window.matchMedia in jsdom:
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false, media: query, onchange: null,
    addListener: () => {}, removeListener: () => {},
    addEventListener: () => {}, removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
})
class ResizeObserverMock { observe(){} unobserve(){} disconnect(){} }
;(globalThis as Record<string, unknown>).ResizeObserver = ResizeObserverMock
class IntersectionObserverMock {
  observe(){} unobserve(){} disconnect(){} takeRecords(){ return [] }
}
;(globalThis as Record<string, unknown>).IntersectionObserver = IntersectionObserverMock
// Some Mantine components call scrollIntoView during interaction.
if (!Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = function () {}
}
