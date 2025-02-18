const themeToggle = document.getElementById('themeToggle');
const body = document.body;

// Load saved theme
const savedTheme = localStorage.getItem('theme');
if (savedTheme) body.setAttribute('data-theme', savedTheme);

themeToggle.addEventListener('click', () => {
  const isDark = body.getAttribute('data-theme') === 'dark';
  body.setAttribute('data-theme', isDark ? 'light' : 'dark');
  localStorage.setItem('theme', isDark ? 'light' : 'dark');
});

function toggleTokenInput(checkbox) {
  const tokenInput = document.getElementById('tokenInput');
  if (checkbox.checked) {
    tokenInput.classList.add('visible');
    localStorage.setItem('tokenVisible', 'true');
  } else {
    tokenInput.classList.remove('visible');
    localStorage.removeItem('tokenVisible');
  }
}

// Restore state on page load
window.addEventListener('DOMContentLoaded', () => {
  const tokenVisible = localStorage.getItem('tokenVisible');
  const checkbox = document.getElementById('privateRepo');
  if (tokenVisible) {
    checkbox.checked = true;
    toggleTokenInput(checkbox);
  }
});
