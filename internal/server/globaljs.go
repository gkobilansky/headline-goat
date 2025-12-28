package server

import (
	"fmt"
	"net/http"
)

// handleGlobalJS serves the global headline-goat script
func (s *Server) handleGlobalJS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Determine server URL from request
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	serverURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	script := GenerateGlobalScript(serverURL)

	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Write([]byte(script))
}

// GenerateGlobalScript generates the global ht.js script with the given server URL
func GenerateGlobalScript(serverURL string) string {
	return fmt.Sprintf(`(function(){
  var S='%s';

  // Get or create visitor ID
  var vid=localStorage.getItem('ht_vid');
  if(!vid){
    vid=crypto.randomUUID();
    localStorage.setItem('ht_vid',vid);
  }

  // Process all test elements
  document.querySelectorAll('[data-ht-name]').forEach(function(el){
    var name=el.dataset.htName;
    var variants=JSON.parse(el.dataset.htVariants||'[]');
    if(!variants.length)return;

    // Check for SSR-selected variant
    if(el.dataset.htSelected!==undefined){
      var selected=parseInt(el.dataset.htSelected);
      beacon(name,selected,'view');
      return;
    }

    // Get or assign variant
    var key='ht_'+name;
    var v=localStorage.getItem(key);
    if(v===null){
      v=Math.floor(Math.random()*variants.length);
      localStorage.setItem(key,v);
    }else{
      v=parseInt(v);
    }

    // Swap text
    el.textContent=variants[v];

    // Send view beacon
    beacon(name,v,'view');
  });

  // Process convert elements
  document.querySelectorAll('[data-ht-convert]').forEach(function(el){
    var name=el.dataset.htConvert;
    var v=parseInt(localStorage.getItem('ht_'+name)||'0');

    // Swap text if variants provided
    var variants=el.dataset.htConvertVariants;
    if(variants){
      variants=JSON.parse(variants);
      if(variants[v])el.textContent=variants[v];
    }

    // URL type: beacon on load
    if(el.dataset.htConvertType==='url'){
      beacon(name,v,'convert');
      return;
    }

    // Click handler
    el.addEventListener('click',function(){
      beacon(name,v,'convert');
    });
  });

  function beacon(t,v,e){
    navigator.sendBeacon(S+'/b',JSON.stringify({t:t,v:v,e:e,vid:vid}));
  }
})();`, serverURL)
}
