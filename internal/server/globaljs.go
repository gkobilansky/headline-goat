package server

import (
	"fmt"
	"net/http"
)

// handleGlobalJS serves the global headline-goat script
func (s *Server) handleGlobalJS(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
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

// GenerateGlobalScript generates the global hlg.js script with the given server URL
func GenerateGlobalScript(serverURL string) string {
	return fmt.Sprintf(`(function(){
  var S='%s';

  // Get or create visitor ID
  var vid=localStorage.getItem('hlg_vid');
  if(!vid){
    vid=crypto.randomUUID();
    localStorage.setItem('hlg_vid',vid);
  }

  // Process all data-attribute test elements (client-side tests)
  document.querySelectorAll('[data-hlg-name]').forEach(function(el){
    var name=el.dataset.hlgName;
    var variants=JSON.parse(el.dataset.hlgVariants||'[]');
    if(!variants.length)return;

    // Check for SSR-selected variant
    if(el.dataset.hlgSelected!==undefined){
      var selected=parseInt(el.dataset.hlgSelected);
      beacon(name,selected,'view',variants,'client');
      return;
    }

    // Get or assign variant
    var key='hlg_'+name;
    var v=localStorage.getItem(key);
    if(v===null){
      v=Math.floor(Math.random()*variants.length);
      localStorage.setItem(key,v);
    }else{
      v=parseInt(v);
    }

    // Swap text
    el.textContent=variants[v];

    // Send view beacon with variants for auto-creation
    beacon(name,v,'view',variants,'client');
  });

  // Process convert elements
  document.querySelectorAll('[data-hlg-convert]').forEach(function(el){
    var name=el.dataset.hlgConvert;
    var v=parseInt(localStorage.getItem('hlg_'+name)||'0');

    // Swap text if variants provided
    var variants=el.dataset.hlgConvertVariants;
    if(variants){
      variants=JSON.parse(variants);
      if(variants[v])el.textContent=variants[v];
    }

    // URL type: beacon on load
    if(el.dataset.hlgConvertType==='url'){
      beacon(name,v,'convert',null,'client');
      return;
    }

    // Click handler
    el.addEventListener('click',function(){
      beacon(name,v,'convert',null,'client');
    });
  });

  // Server-side test handling with cache
  (function(){
    var path=location.pathname;
    var cacheKey='hlg_tests_'+path;
    var cached=localStorage.getItem(cacheKey);

    // Apply cached tests immediately (no flash on repeat visits)
    if(cached){
      try{
        applyServerTests(JSON.parse(cached));
      }catch(e){}
    }

    // Fetch fresh config in background, update cache
    fetch(S+'/api/tests?url='+encodeURIComponent(path))
      .then(function(r){return r.json()})
      .then(function(tests){
        // Update cache for next visit
        localStorage.setItem(cacheKey,JSON.stringify(tests));
        // Apply if not already applied from cache
        if(!cached)applyServerTests(tests);
      })
      .catch(function(){});
  })();

  function applyServerTests(tests){
    if(!tests||!tests.length)return;
    tests.forEach(function(test){
      // Skip if already processed via data attributes
      if(document.querySelector('[data-hlg-name="'+test.name+'"]'))return;

      // Find target element
      if(!test.target)return;
      var el=document.querySelector(test.target);
      if(!el)return;

      // Assign variant (same localStorage pattern)
      var key='hlg_'+test.name;
      var v=localStorage.getItem(key);
      if(v===null){
        v=Math.floor(Math.random()*test.variants.length);
        localStorage.setItem(key,v);
      }else{
        v=parseInt(v);
      }

      // Apply variant
      if(test.variants[v])el.textContent=test.variants[v];
      beacon(test.name,v,'view',null,'server');

      // Setup conversion tracking
      if(test.cta_target){
        var cta=document.querySelector(test.cta_target);
        if(cta){
          cta.addEventListener('click',function(){
            beacon(test.name,v,'convert',null,'server');
          });
        }
      }
      if(test.conversion_url&&location.pathname===test.conversion_url){
        beacon(test.name,v,'convert',null,'server');
      }
    });
  }

  function beacon(t,v,e,variants,src){
    var payload={t:t,v:v,e:e,vid:vid,src:src||'client'};
    if(variants)payload.variants=variants;
    navigator.sendBeacon(S+'/b',JSON.stringify(payload));
  }
})();`, serverURL)
}
