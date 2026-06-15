const darkModeSwitch = document.getElementById('darkModeSwitch');
const body = document.body;

// Apply theme classes immediately (data-bs-theme already set by inline head script)
const savedMode = localStorage.getItem('dark-mode');
if (savedMode === 'true') {
  body.classList.add('dark-mode');
  body.classList.remove('light-mode');
  darkModeSwitch.checked = true;
} else {
  body.classList.add('light-mode');
  body.classList.remove('dark-mode');
  darkModeSwitch.checked = false;
}

darkModeSwitch.addEventListener('change', () => {
  const isDarkMode = darkModeSwitch.checked;
  body.classList.toggle('dark-mode', isDarkMode);
  body.classList.toggle('light-mode', !isDarkMode);
  document.documentElement.setAttribute('data-bs-theme', isDarkMode ? 'dark' : 'light');
  document.documentElement.style.colorScheme = isDarkMode ? 'dark' : 'light';
  localStorage.setItem('dark-mode', isDarkMode);
});
