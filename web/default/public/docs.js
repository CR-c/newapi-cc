;(() => {
  const sectionNav = document.querySelector('.section-nav')
  if (!sectionNav) return

  const links = Array.from(sectionNav.querySelectorAll('a[href^="#"]'))
  if (!links.length) return

  const sections = links
    .map((link) => {
      const raw = link.getAttribute('href') || ''
      const id = decodeURIComponent(raw.replace(/^#/, ''))
      if (!id) return null
      const el = document.getElementById(id)
      return el ? { id, link, el } : null
    })
    .filter(Boolean)

  if (!sections.length) return

  let activeId = null
  // Keep hash/click selection for a short time so browser scroll-to-anchor
  // and IntersectionObserver do not flash the wrong chapter.
  let preferHashUntil = 0

  function stickyOffset() {
    const sticky = document.querySelector('.docs-sticky')
    return (sticky?.offsetHeight || 96) + 12
  }

  function setActive(id, { force = false } = {}) {
    if (!id) return
    if (!force && id === activeId) return
    if (!sections.some((s) => s.id === id)) return

    activeId = id
    for (const item of sections) {
      const on = item.id === id
      item.link.classList.toggle('is-active', on)
      if (on) {
        item.link.setAttribute('aria-current', 'location')
      } else {
        item.link.removeAttribute('aria-current')
      }
    }

    const activeLink = sections.find((s) => s.id === id)?.link
    if (!activeLink || typeof activeLink.scrollIntoView !== 'function') return

    const linkRect = activeLink.getBoundingClientRect()
    const navRect = sectionNav.getBoundingClientRect()
    if (linkRect.left < navRect.left + 8 || linkRect.right > navRect.right - 8) {
      activeLink.scrollIntoView({
        behavior: 'smooth',
        block: 'nearest',
        inline: 'center',
      })
    }
  }

  function pickFromScroll() {
    if (Date.now() < preferHashUntil) return

    const offset = stickyOffset()
    let current = sections[0].id
    for (const item of sections) {
      const top = item.el.getBoundingClientRect().top
      if (top - offset <= 1) {
        current = item.id
      } else {
        break
      }
    }
    setActive(current)
  }

  function hashId() {
    const raw = location.hash || ''
    if (!raw || raw === '#') return ''
    try {
      return decodeURIComponent(raw.replace(/^#/, ''))
    } catch {
      return raw.replace(/^#/, '')
    }
  }

  function applyHash(preferMs = 900) {
    const id = hashId()
    if (id && sections.some((s) => s.id === id)) {
      preferHashUntil = Date.now() + preferMs
      setActive(id, { force: true })
      return true
    }
    return false
  }

  for (const item of sections) {
    item.link.addEventListener('click', () => {
      preferHashUntil = Date.now() + 900
      setActive(item.id, { force: true })
    })
  }

  window.addEventListener(
    'scroll',
    () => {
      window.requestAnimationFrame(pickFromScroll)
    },
    { passive: true }
  )
  window.addEventListener('resize', () => {
    window.requestAnimationFrame(pickFromScroll)
  })
  window.addEventListener('hashchange', () => {
    if (!applyHash(900)) pickFromScroll()
  })

  if ('IntersectionObserver' in window) {
    const rootMarginTop = -stickyOffset()
    const observer = new IntersectionObserver(
      () => {
        // Always resolve via scroll position so multi-visible sections stay stable.
        pickFromScroll()
      },
      {
        root: null,
        rootMargin: `${rootMarginTop}px 0px -45% 0px`,
        threshold: [0, 0.05, 0.15, 0.35, 0.6, 1],
      }
    )
    for (const item of sections) observer.observe(item.el)
  }

  // Initial: honor #hash first (e.g. /docs-official.html#start), else first section.
  if (!applyHash(1200)) {
    setActive(sections[0].id, { force: true })
    // After layout/hash jump settles, re-sync once.
    window.setTimeout(() => {
      if (!applyHash(0)) pickFromScroll()
    }, 50)
    window.setTimeout(pickFromScroll, 200)
  } else {
    window.setTimeout(() => {
      // Re-apply hash after browser scrolls to anchor.
      if (!applyHash(600)) pickFromScroll()
    }, 80)
    window.setTimeout(pickFromScroll, 700)
  }
})()
