// Custom JS injection test
console.log('W2App: Custom JS injected successfully!');

// Add custom notification
document.addEventListener('DOMContentLoaded', function() {
    setTimeout(function() {
        var notice = document.createElement('div');
        notice.style.cssText = 'position:fixed;top:10px;right:10px;background:#4CAF50;color:white;padding:10px 20px;border-radius:5px;z-index:99999;font-family:sans-serif;';
        notice.textContent = 'W2App Desktop Mode';
        document.body.appendChild(notice);
        setTimeout(function() { notice.remove(); }, 3000);
    }, 1000);
});
