// Wait for the document to be fully loaded
document.addEventListener('DOMContentLoaded', function() {
  // Add copy buttons to code blocks
  addCopyButtons();
  
  // Add dark mode toggle
  setupDarkModeToggle();
  
  // Add scroll to top button
  addScrollToTopButton();
  
  // Add code highlighting
  highlightCurrentSection();
});

// Function to add copy buttons to code blocks
function addCopyButtons() {
  const codeBlocks = document.querySelectorAll('pre code');
  
  codeBlocks.forEach((codeBlock) => {
    // Skip if already has a copy button
    if (codeBlock.parentNode.querySelector('.copy-button')) {
      return;
    }
    
    const button = document.createElement('button');
    button.className = 'copy-button';
    button.textContent = 'Copy';
    
    button.addEventListener('click', () => {
      const code = codeBlock.textContent;
      navigator.clipboard.writeText(code).then(() => {
        button.textContent = 'Copied!';
        setTimeout(() => {
          button.textContent = 'Copy';
        }, 2000);
      });
    });
    
    // Insert button before code block
    codeBlock.parentNode.insertBefore(button, codeBlock);
  });
}

// Function to setup dark mode toggle
function setupDarkModeToggle() {
  const toggle = document.querySelector('.md-header__button[for="__palette"]');
  if (toggle) {
    toggle.addEventListener('click', () => {
      document.documentElement.classList.toggle('dark-mode');
    });
  }
}

// Function to add scroll to top button
function addScrollToTopButton() {
  const button = document.createElement('button');
  button.className = 'scroll-to-top';
  button.innerHTML = '↑';
  button.title = 'Scroll to top';
  
  button.addEventListener('click', () => {
    window.scrollTo({
      top: 0,
      behavior: 'smooth'
    });
  });
  
  document.body.appendChild(button);
  
  // Show/hide button based on scroll position
  window.addEventListener('scroll', () => {
    if (window.scrollY > 300) {
      button.classList.add('visible');
    } else {
      button.classList.remove('visible');
    }
  });
}

// Function to highlight the current section in the navigation
function highlightCurrentSection() {
  const currentPath = window.location.pathname;
  const navLinks = document.querySelectorAll('.md-nav__link');
  
  navLinks.forEach(link => {
    const linkPath = link.getAttribute('href');
    if (linkPath && currentPath.endsWith(linkPath)) {
      link.classList.add('md-nav__link--active');
    }
  });
} 