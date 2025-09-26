async function load(){
  const res = await fetch('/api/items');
  const items = await res.json();
  const ul = document.getElementById('list');
  ul.innerHTML = '';
  for (const it of items){
    const li = document.createElement('li');
    li.textContent = `${it.id}. ${it.title} — ${it.note||''}`;
    ul.appendChild(li);
  }
}

const f = document.getElementById('f');
f.addEventListener('submit', async (e)=>{
  e.preventDefault();
  const title = document.getElementById('title').value.trim();
  const note = document.getElementById('note').value.trim();
  if(!title) return;
  await fetch('/api/items', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({title, note})});
  document.getElementById('title').value='';
  document.getElementById('note').value='';
  load();
});

load();
