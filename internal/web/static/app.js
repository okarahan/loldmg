const palette = [
  '#38bdf8', '#f472b6', '#a78bfa', '#34d399', '#fbbf24',
  '#fb7185', '#60a5fa', '#c084fc', '#2dd4bf', '#f87171'
];

const search = document.getElementById('search');
const errorEl = document.getElementById('error');
const loadingEl = document.getElementById('loading');
const timelineWrap = document.getElementById('timelineWrap');
const timeline = document.getElementById('timeline');
const fightPanel = document.getElementById('fightPanel');
const fightTitle = document.getElementById('fightTitle');
const fightMeter = document.getElementById('fightMeter');

let currentData = null;

search.addEventListener('submit', async (e) => {
  e.preventDefault();
  const matchId = document.getElementById('matchId').value.trim();
  const region = document.getElementById('region').value;

  errorEl.classList.add('hidden');
  timelineWrap.classList.add('hidden');
  fightPanel.classList.add('hidden');
  loadingEl.classList.remove('hidden');

  try {
    const res = await fetch(`/api/analyze?match_id=${encodeURIComponent(matchId)}&region=${region}`);
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || `HTTP ${res.status}`);
    }
    currentData = data;
    renderTimeline(data);
    timelineWrap.classList.remove('hidden');
  } catch (err) {
    errorEl.textContent = err.message;
    errorEl.classList.remove('hidden');
  } finally {
    loadingEl.classList.add('hidden');
  }
});

function formatTime(ms) {
  const totalSec = Math.floor(ms / 1000);
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}

function renderTimeline(data) {
  timeline.innerHTML = '';
  if (!data.fights || data.fights.length === 0) {
    timeline.innerHTML = '<div style="padding:1rem;color:var(--muted)">No fights detected.</div>';
    return;
  }

  const start = Math.min(...data.fights.map(f => f.start_ms));
  const end = Math.max(...data.fights.map(f => f.end_ms));
  const duration = Math.max(end - start, 1);

  // Kill density buckets
  const buckets = 60;
  const kills = [];
  data.fights.forEach(f => f.kill_events.forEach(k => kills.push(k.timestamp)));
  const bucketCounts = new Array(buckets).fill(0);
  kills.forEach(t => {
    const idx = Math.min(buckets - 1, Math.floor(((t - start) / duration) * buckets));
    bucketCounts[idx]++;
  });
  const maxCount = Math.max(1, ...bucketCounts);

  bucketCounts.forEach((count, i) => {
    const div = document.createElement('div');
    div.className = 'timeline-bucket';
    div.style.left = `${(i / buckets) * 100}%`;
    div.style.width = `${100 / buckets}%`;
    div.style.background = `rgba(239, 68, 68, ${count / maxCount * 0.7})`;
    timeline.appendChild(div);
  });

  // Objective markers from fight labels and kill events
  const objectives = [];
  data.fights.forEach(f => {
    f.kill_events.forEach(k => {
      if (k.type === 'ELITE_MONSTER_KILL') {
        objectives.push({ t: k.timestamp, kind: monsterKind(k.monster_type) });
      } else if (k.type === 'BUILDING_KILL') {
        objectives.push({ t: k.timestamp, kind: buildingKind(k.building_type) });
      }
    });
  });
  objectives.forEach(o => {
    const div = document.createElement('div');
    div.className = `timeline-objective ${o.kind.class}`;
    div.style.left = `${((o.t - start) / duration) * 100}%`;
    div.title = o.kind.label;
    div.textContent = o.kind.short;
    timeline.appendChild(div);
  });

  // Fight blocks
  data.fights.forEach((f, idx) => {
    const div = document.createElement('div');
    div.className = 'timeline-fight';
    div.style.left = `${((f.start_ms - start) / duration) * 100}%`;
    div.style.width = `${Math.max(1, (f.end_ms - f.start_ms) / duration * 100)}%`;
    div.textContent = f.label || 'Fight';
    div.title = `${formatTime(f.start_ms)} - ${formatTime(f.end_ms)} | ${f.kills} kill(s)`;
    div.addEventListener('click', () => showFight(f, idx, data.info));
    timeline.appendChild(div);
  });
}

function monsterKind(type) {
  switch (type) {
    case 'BARON_NASHOR': return { class: 'baron', label: 'Baron Nashor', short: 'B' };
    case 'ELDER_DRAGON': return { class: 'dragon', label: 'Elder Dragon', short: 'E' };
    case 'DRAGON': return { class: 'dragon', label: 'Dragon', short: 'D' };
    case 'RIFT_HERALD': return { class: 'herald', label: 'Rift Herald', short: 'H' };
    case 'ATAKHAN': return { class: 'atakhan', label: 'Atakhan', short: 'A' };
    default: return { class: 'dragon', label: type, short: '?' };
  }
}

function buildingKind(type) {
  switch (type) {
    case 'INHIBITOR_BUILDING': return { class: 'inhibitor', label: 'Inhibitor', short: 'I' };
    case 'TOWER_BUILDING': return { class: 'tower', label: 'Tower', short: 'T' };
    default: return { class: 'tower', label: type, short: '?' };
  }
}

function showFight(fight, idx, info) {
  fightPanel.classList.remove('hidden');
  fightTitle.textContent = `Fight ${idx + 1} — ${fight.label || 'Skirmish'} @ ${formatTime(fight.start_ms)} - ${formatTime(fight.end_ms)} (${fight.kills} kills)`;
  fightMeter.innerHTML = '';

  const identities = {};
  if (info && info.info && info.info.participants) {
    info.info.participants.forEach(p => {
      identities[p.participantId] = p.championName || `P${p.participantId}`;
    });
  }

  const entries = Object.entries(fight.damage)
    .map(([pid, dmg]) => {
      const id = parseInt(pid, 10);
      const name = identities[id] || `P${pid}`;
      const kills = (fight.kills_by_player[pid] || 0);
      const assists = (fight.assists_by_player[pid] || 0);
      const deaths = (fight.deaths_by_player[pid] || 0);
      return { id, name, dmg, kda: `${kills}/${deaths}/${assists}` };
    })
    .sort((a, b) => b.dmg - a.dmg);

  const maxDmg = Math.max(1, entries[0]?.dmg || 1);

  entries.forEach((e, i) => {
    const row = document.createElement('div');
    row.className = 'fight-player';

    const color = palette[i % palette.length];
    const pct = (e.dmg / maxDmg) * 100;

    row.innerHTML = `
      <div class="name" title="${e.name}">${e.name} <span class="kda">${e.kda}</span></div>
      <div class="bar-wrap">
        <div class="bar" style="width: ${pct}%; background: ${color};"
             title="${e.dmg} damage">${e.dmg}</div>
      </div>
      <div class="value">${e.dmg}</div>
    `;
    fightMeter.appendChild(row);
  });
}
