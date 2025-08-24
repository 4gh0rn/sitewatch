/**
 * SiteWatch Core JavaScript Library
 * Centralized error handling, loading states, and utilities
 */

window.SiteWatch = {
    // Configuration
    config: {
        retryAttempts: 3,
        retryDelay: 1000,
        loadingTimeout: 30000,
        debugMode: false
    },
    
    // Error handling
    errors: {
        network: 'Network connection failed. Please check your internet connection.',
        timeout: 'Request timed out. Please try again.',
        server: 'Server error occurred. Please try again later.',
        notFound: 'Requested resource not found.',
        generic: 'An unexpected error occurred. Please try again.'
    },
    
    // Initialize core functionality
    init() {
        this.setupGlobalErrorHandling();
        this.setupNetworkMonitoring();
        this.setupRetryMechanism();
        
        if (this.config.debugMode) {
            console.log('🚀 SiteWatch Core initialized');
        }
    },
    
    // Global error handling setup
    setupGlobalErrorHandling() {
        // Handle unhandled JavaScript errors
        window.addEventListener('error', (event) => {
            this.logError('JavaScript Error', {
                message: event.message,
                filename: event.filename,
                line: event.lineno,
                column: event.colno
            });
        });
        
        // Handle unhandled promise rejections
        window.addEventListener('unhandledrejection', (event) => {
            this.logError('Unhandled Promise Rejection', {
                reason: event.reason
            });
            event.preventDefault();
        });
        
        // Handle HTMX errors if HTMX is available
        if (typeof htmx !== 'undefined') {
            document.addEventListener('htmx:responseError', (event) => {
                this.handleHTMXError(event);
            });
            
            document.addEventListener('htmx:sendError', (event) => {
                this.handleHTMXError(event);
            });
        }
    },
    
    // Network monitoring
    setupNetworkMonitoring() {
        if ('navigator' in window && 'onLine' in navigator) {
            window.addEventListener('online', () => {
                this.showNotification('Connection restored', 'success');
            });
            
            window.addEventListener('offline', () => {
                this.showNotification('Connection lost', 'warning');
            });
        }
    },
    
    // Retry mechanism for failed requests
    setupRetryMechanism() {
        this.retryQueue = new Map();
    },
    
    // Enhanced loading state management
    showLoading(element, message = 'Loading...', showSpinner = true) {
        if (!element) return;
        
        // Clear any existing content
        element.innerHTML = '';
        element.className = 'loading-state flex items-center justify-center p-8 bg-gray-50 rounded-lg';
        
        const loadingContent = document.createElement('div');
        loadingContent.className = 'text-center';
        
        if (showSpinner) {
            const spinner = document.createElement('div');
            spinner.className = 'animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto mb-4';
            loadingContent.appendChild(spinner);
        }
        
        const text = document.createElement('p');
        text.className = 'text-gray-600';
        text.textContent = message;
        loadingContent.appendChild(text);
        
        element.appendChild(loadingContent);
        element.setAttribute('aria-live', 'polite');
        element.setAttribute('aria-busy', 'true');
    },
    
    // Enhanced error state management
    showError(element, message, allowRetry = true) {
        if (!element) return;
        
        element.innerHTML = '';
        element.className = 'error-state p-6 bg-red-50 border border-red-200 rounded-lg';
        
        const errorContent = document.createElement('div');
        errorContent.className = 'text-center';
        
        // Error icon
        const icon = document.createElement('div');
        icon.className = 'mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-red-100 mb-4';
        icon.innerHTML = `
            <svg class="h-6 w-6 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.732-.833-2.5 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
        `;
        errorContent.appendChild(icon);
        
        // Error message
        const text = document.createElement('p');
        text.className = 'text-red-800 font-medium mb-4';
        text.textContent = message;
        errorContent.appendChild(text);
        
        // Retry button if allowed
        if (allowRetry) {
            const retryButton = document.createElement('button');
            retryButton.className = 'btn btn-outline-red touch-target';
            retryButton.textContent = 'Try Again';
            retryButton.addEventListener('click', () => {
                this.retryLastAction(element);
            });
            errorContent.appendChild(retryButton);
        }
        
        element.appendChild(errorContent);
        element.setAttribute('aria-live', 'assertive');
        element.setAttribute('aria-busy', 'false');
    },
    
    // Success state
    showSuccess(element, message, autoHide = true) {
        if (!element) return;
        
        element.innerHTML = '';
        element.className = 'success-state p-6 bg-green-50 border border-green-200 rounded-lg';
        
        const successContent = document.createElement('div');
        successContent.className = 'text-center';
        
        // Success icon
        const icon = document.createElement('div');
        icon.className = 'mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-green-100 mb-4';
        icon.innerHTML = `
            <svg class="h-6 w-6 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
        `;
        successContent.appendChild(icon);
        
        // Success message
        const text = document.createElement('p');
        text.className = 'text-green-800 font-medium';
        text.textContent = message;
        successContent.appendChild(text);
        
        element.appendChild(successContent);
        element.setAttribute('aria-live', 'polite');
        
        if (autoHide) {
            setTimeout(() => {
                if (element.classList.contains('success-state')) {
                    element.classList.add('fade-out');
                    setTimeout(() => element.remove(), 300);
                }
            }, 3000);
        }
    },
    
    // Enhanced notification system
    showNotification(message, type = 'info', duration = 5000) {
        const notification = document.createElement('div');
        notification.className = `notification notification-${type} fixed top-4 right-4 z-50 max-w-sm`;
        
        const colors = {
            success: 'bg-green-500',
            error: 'bg-red-500',
            warning: 'bg-yellow-500',
            info: 'bg-blue-500'
        };
        
        notification.innerHTML = `
            <div class="${colors[type]} text-white p-4 rounded-lg shadow-lg">
                <div class="flex items-center">
                    <span class="flex-1">${message}</span>
                    <button class="ml-4 text-white hover:text-gray-200" onclick="this.parentElement.parentElement.remove()">
                        <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                            <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"/>
                        </svg>
                    </button>
                </div>
            </div>
        `;
        
        document.body.appendChild(notification);
        
        // Auto-remove after duration
        if (duration > 0) {
            setTimeout(() => {
                if (notification.parentElement) {
                    notification.classList.add('fade-out');
                    setTimeout(() => notification.remove(), 300);
                }
            }, duration);
        }
        
        return notification;
    },
    
    // Enhanced fetch with retry and error handling
    async fetchWithRetry(url, options = {}, retries = this.config.retryAttempts) {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.config.loadingTimeout);
        
        try {
            const response = await fetch(url, {
                ...options,
                signal: controller.signal
            });
            
            clearTimeout(timeoutId);
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            return response;
        } catch (error) {
            clearTimeout(timeoutId);
            
            if (retries > 0 && !controller.signal.aborted) {
                await new Promise(resolve => setTimeout(resolve, this.config.retryDelay));
                return this.fetchWithRetry(url, options, retries - 1);
            }
            
            throw error;
        }
    },
    
    // HTMX error handling
    handleHTMXError(event) {
        const statusCode = event.detail.xhr?.status;
        let message = this.errors.generic;
        
        switch (statusCode) {
            case 0:
                message = this.errors.network;
                break;
            case 404:
                message = this.errors.notFound;
                break;
            case 500:
            case 502:
            case 503:
                message = this.errors.server;
                break;
            case 408:
                message = this.errors.timeout;
                break;
        }
        
        this.showError(event.target, message);
    },
    
    // Retry last action
    retryLastAction(element) {
        const htmxAttributes = ['hx-get', 'hx-post', 'hx-put', 'hx-delete'];
        
        for (const attr of htmxAttributes) {
            if (element.hasAttribute(attr)) {
                this.showLoading(element, 'Retrying...');
                if (typeof htmx !== 'undefined') {
                    htmx.trigger(element, 'retry');
                }
                break;
            }
        }
    },
    
    // Error logging
    logError(type, details) {
        const errorInfo = {
            type,
            timestamp: new Date().toISOString(),
            userAgent: navigator.userAgent,
            url: window.location.href,
            ...details
        };
        
        if (this.config.debugMode) {
            console.error('SiteWatch Error:', errorInfo);
        }
        
        // You could send this to your logging service here
    },
    
    // Utility functions
    utils: {
        // Format time ago
        timeAgo(date) {
            const now = new Date();
            const diffInSeconds = Math.floor((now - date) / 1000);
            
            if (diffInSeconds < 60) return 'just now';
            if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m ago`;
            if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h ago`;
            return `${Math.floor(diffInSeconds / 86400)}d ago`;
        },
        
        // Format bytes
        formatBytes(bytes, decimals = 2) {
            if (bytes === 0) return '0 Bytes';
            
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            
            return parseFloat((bytes / Math.pow(k, i)).toFixed(decimals)) + ' ' + sizes[i];
        },
        
        // Debounce function
        debounce(func, wait) {
            let timeout;
            return function executedFunction(...args) {
                const later = () => {
                    clearTimeout(timeout);
                    func(...args);
                };
                clearTimeout(timeout);
                timeout = setTimeout(later, wait);
            };
        },
        
        // Throttle function
        throttle(func, limit) {
            let inThrottle;
            return function() {
                const args = arguments;
                const context = this;
                if (!inThrottle) {
                    func.apply(context, args);
                    inThrottle = true;
                    setTimeout(() => inThrottle = false, limit);
                }
            };
        }
    }
};

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    window.SiteWatch.init();
});

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = window.SiteWatch;
}