const darkModeSwitch = document.getElementById('darkModeSwitch');
const body = document.body;

// Check for saved dark mode preference and apply early
const savedMode = localStorage.getItem('dark-mode');
if (savedMode === 'true') {
    body.classList.add('dark-mode');
    darkModeSwitch.checked = true;
} else {
    body.classList.add('light-mode');
    darkModeSwitch.checked = false;
}

// Load page with content hidden to prevent flash of unstyled content
window.addEventListener('load', () => {
  // Apply dark mode or light mode after content is loaded
  if (savedMode === 'true') {
    body.classList.add('dark-mode');
    darkModeSwitch.checked = true;
  } else {
    body.classList.add('light-mode');
  }

  // Make the body visible after applying the mode
  body.style.visibility = 'visible';
});

darkModeSwitch.addEventListener('change', () => {
    const isDarkMode = darkModeSwitch.checked;
    body.classList.toggle('dark-mode', isDarkMode);
    body.classList.toggle('light-mode', !isDarkMode);
    localStorage.setItem('dark-mode', isDarkMode);
});
