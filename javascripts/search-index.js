/**
 * Enhanced search index generator for Blink documentation
 * This script improves the default MkDocs search functionality
 */

document.addEventListener('DOMContentLoaded', function() {
  // Wait for the search plugin to initialize
  setTimeout(enhanceSearch, 1000);
});

function enhanceSearch() {
  // Get the search input element
  const searchInput = document.querySelector('.md-search__input');
  if (!searchInput) return;
  
  // Add placeholder text
  searchInput.setAttribute('placeholder', 'Search Blink docs...');
  
  // Add event listener for search input
  searchInput.addEventListener('focus', function() {
    // Add search suggestions
    addSearchSuggestions();
  });
  
  // Enhance search results display
  enhanceSearchResults();
}

function addSearchSuggestions() {
  // Common search terms
  const suggestions = [
    'installation',
    'configuration',
    'webhooks',
    'event filtering',
    'performance',
    'CLI',
    'API reference'
  ];
  
  // Get or create suggestions container
  let suggestionsContainer = document.querySelector('.search-suggestions');
  if (!suggestionsContainer) {
    suggestionsContainer = document.createElement('div');
    suggestionsContainer.className = 'search-suggestions';
    
    // Add title
    const title = document.createElement('div');
    title.className = 'search-suggestions__title';
    title.textContent = 'Common searches:';
    suggestionsContainer.appendChild(title);
    
    // Add suggestions
    const list = document.createElement('div');
    list.className = 'search-suggestions__list';
    
    suggestions.forEach(suggestion => {
      const item = document.createElement('button');
      item.className = 'search-suggestions__item';
      item.textContent = suggestion;
      item.addEventListener('click', () => {
        const searchInput = document.querySelector('.md-search__input');
        if (searchInput) {
          searchInput.value = suggestion;
          searchInput.dispatchEvent(new Event('input'));
        }
      });
      list.appendChild(item);
    });
    
    suggestionsContainer.appendChild(list);
    
    // Add to search modal
    const searchModal = document.querySelector('.md-search__inner');
    if (searchModal) {
      searchModal.appendChild(suggestionsContainer);
    }
  }
}

function enhanceSearchResults() {
  // Create a mutation observer to watch for search results
  const observer = new MutationObserver(mutations => {
    mutations.forEach(mutation => {
      if (mutation.addedNodes && mutation.addedNodes.length > 0) {
        // Check if search results are added
        const searchResults = document.querySelector('.md-search-result');
        if (searchResults) {
          // Add category labels to search results
          addCategoryLabels(searchResults);
        }
      }
    });
  });
  
  // Start observing the document body for changes
  observer.observe(document.body, { childList: true, subtree: true });
}

function addCategoryLabels(searchResults) {
  // Get all search result items
  const items = searchResults.querySelectorAll('.md-search-result__item');
  
  items.forEach(item => {
    // Get the article element
    const article = item.querySelector('article');
    if (!article) return;
    
    // Get the location (breadcrumbs)
    const location = item.querySelector('.md-search-result__terms');
    if (!location) return;
    
    // Determine category based on location text
    let category = 'General';
    const locationText = location.textContent.toLowerCase();
    
    if (locationText.includes('api')) {
      category = 'API';
    } else if (locationText.includes('getting started')) {
      category = 'Getting Started';
    } else if (locationText.includes('features')) {
      category = 'Features';
    } else if (locationText.includes('advanced')) {
      category = 'Advanced';
    } else if (locationText.includes('usage')) {
      category = 'Usage';
    }
    
    // Create category label if it doesn't exist
    if (!item.querySelector('.search-category')) {
      const categoryLabel = document.createElement('span');
      categoryLabel.className = 'search-category';
      categoryLabel.textContent = category;
      categoryLabel.style.display = 'inline-block';
      categoryLabel.style.padding = '2px 6px';
      categoryLabel.style.borderRadius = '4px';
      categoryLabel.style.fontSize = '0.7rem';
      categoryLabel.style.marginRight = '8px';
      categoryLabel.style.backgroundColor = getCategoryColor(category);
      categoryLabel.style.color = 'white';
      
      // Add to the article
      article.insertBefore(categoryLabel, article.firstChild);
    }
  });
}

function getCategoryColor(category) {
  // Return color based on category
  switch (category) {
    case 'API':
      return '#e91e63';
    case 'Getting Started':
      return '#4caf50';
    case 'Features':
      return '#ff9800';
    case 'Advanced':
      return '#9c27b0';
    case 'Usage':
      return '#03a9f4';
    default:
      return '#757575';
  }
} 