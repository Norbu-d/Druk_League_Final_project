// DrukLeague landing page.
// An empty hash or #home shows the landing page; any other hash (#dashboard, #teams, ...)
// shows the workspace rendered by app.js. app.js asks window.DrukLanding.active() before
// rendering so it does not change the URL or scroll position while the landing page is open.
(() => {
  const landing = document.getElementById('landing');
  const shell = document.querySelector('.app-shell');
  if (!landing || !shell) return;

  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  const isLandingHash = () => ['', '#', '#home'].includes(location.hash);
  window.DrukLanding = Object.freeze({ active: isLandingHash });

  landing.classList.add('lp-js');

  function syncView() {
    const show = isLandingHash();
    landing.hidden = !show;
    shell.hidden = show;
    document.body.classList.toggle('landing-active', show);
    if (show) {
      updateSignInLabel();
      window.scrollTo(0, 0);
    }
  }

  function updateSignInLabel() {
    const signedIn = Boolean(window.DRUKLEAGUE_AUTHENTICATED);
    landing.querySelectorAll('[data-lp-signin]').forEach((button) => {
      button.hidden = signedIn;
    });
  }

  // mode: 'open' (workspace only), 'signin' or 'signup' (account screen for signed-out visitors).
  function openWorkspace(mode) {
    location.hash = '#dashboard';
    if (mode !== 'open' && !window.DRUKLEAGUE_AUTHENTICATED) window.DrukApp?.openSignIn(mode);
  }

  // Buttons: open the workspace, sign in, or scroll to a landing section.
  landing.addEventListener('click', (event) => {
    const action = event.target.closest('[data-lp-action]');
    if (action) {
      openWorkspace(action.dataset.lpAction);
      return;
    }
    const scroller = event.target.closest('[data-lp-scroll]');
    if (scroller) {
      document.getElementById(scroller.dataset.lpScroll)?.scrollIntoView({ behavior: reduceMotion ? 'auto' : 'smooth', block: 'start' });
    }
  });

  // Hero photo slider: arrows plus a slow automatic advance.
  const slides = [...landing.querySelectorAll('.lp-slide')];
  let current = 0;
  let timer = null;
  function showSlide(index) {
    slides[current].classList.remove('is-active');
    current = (index + slides.length) % slides.length;
    slides[current].classList.add('is-active');
  }
  const hero = landing.querySelector('.lp-hero');
  function restartTimer() {
    if (reduceMotion || slides.length < 2) return;
    window.clearInterval(timer);
    timer = window.setInterval(() => {
      if (!landing.hidden && !hero.classList.contains('lp-paused')) showSlide(current + 1);
    }, 8000);
  }
  landing.querySelectorAll('[data-lp-slide]').forEach((button) => {
    button.addEventListener('click', () => {
      showSlide(current + Number(button.dataset.lpSlide));
      restartTimer();
    });
  });
  restartTimer();

  // Marquees scroll by -50%, so duplicate each track's items once for a seamless loop.
  landing.querySelectorAll('.lp-track, .lp-ticker-track').forEach((track) => {
    [...track.children].forEach((item) => {
      const copy = item.cloneNode(true);
      copy.setAttribute('aria-hidden', 'true');
      copy.querySelectorAll('img').forEach((img) => { img.alt = ''; });
      track.appendChild(copy);
    });
  });

  // Pause looping animations (hero slider and floats, marquees, ticker) while they are off screen.
  if ('IntersectionObserver' in window) {
    const pauser = new IntersectionObserver((entries) => {
      entries.forEach((entry) => entry.target.classList.toggle('lp-paused', !entry.isIntersecting));
    }, { rootMargin: '100px 0px' });
    landing.querySelectorAll('.lp-hero, .lp-marquee, .lp-ticker').forEach((element) => pauser.observe(element));
  }

  // Count a number up from zero when it scrolls into view.
  function countUp(element) {
    const target = Number(element.dataset.lpCount);
    if (!Number.isFinite(target) || reduceMotion) return;
    const start = performance.now();
    const duration = 1400;
    const step = (now) => {
      const progress = Math.min((now - start) / duration, 1);
      element.textContent = String(Math.round(target * (1 - Math.pow(1 - progress, 3))));
      if (progress < 1) requestAnimationFrame(step);
    };
    requestAnimationFrame(step);
  }

  // Reveal sections as they scroll into view, staggering siblings.
  const revealables = [...landing.querySelectorAll('.lp-reveal')];
  revealables.forEach((element) => {
    const siblings = [...element.parentElement.children].filter((child) => child.classList.contains('lp-reveal'));
    element.style.setProperty('--lp-delay', `${Math.min(siblings.indexOf(element), 5) * 0.09}s`);
  });
  const steps = landing.querySelector('.lp-steps');
  if ('IntersectionObserver' in window && !reduceMotion) {
    const observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (!entry.isIntersecting) return;
        const element = entry.target;
        element.classList.add('is-visible');
        if (element === steps) element.classList.add('is-drawn');
        element.querySelectorAll('[data-lp-count]').forEach(countUp);
        observer.unobserve(element);
      });
    }, { threshold: 0.18 });
    revealables.forEach((element) => observer.observe(element));
    if (steps) observer.observe(steps);
  } else {
    revealables.forEach((element) => element.classList.add('is-visible'));
    steps?.classList.add('is-drawn');
  }

  window.addEventListener('hashchange', syncView);
  syncView();
  // The saved session is restored after load; hide "Sign in" once it is known.
  window.setTimeout(updateSignInLabel, 1500);
})();
