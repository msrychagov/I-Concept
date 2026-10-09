// Recheck the session on the server when the browser restores a cached page.
window.addEventListener('pageshow',event=>{
 if(event.persisted)window.location.reload();
});
