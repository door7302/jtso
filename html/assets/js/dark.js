
const darkModeSwitch = document.getElementById('darkModeSwitch');
const body = document.body;

// Load saved mode from localStorage
const savedMode = localStorage.getItem('dark-mode');
if (savedMode === 'true') {
    body.classList.add('dark-mode');
    darkModeSwitch.checked = true;
}

darkModeSwitch.addEventListener('change', () => {
    const isDarkMode = darkModeSwitch.checked;
    body.classList.toggle('dark-mode', isDarkMode);
    localStorage.setItem('dark-mode', isDarkMode);
});