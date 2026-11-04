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

  data.fights.forEach((f, idx) => {
    const kills = f.kill_events ? f.kill_events.length : 0;
    const card = document.createElement('div');
    card.className = 'fight-card';

    const header = document.createElement('div');
    header.className = 'fight-card-header';
    header.textContent = `Fight ${idx + 1}`;
    card.appendChild(header);

    const body = document.createElement('div');
    body.className = 'fight-card-body';
    body.textContent = f.label || 'Skirmish';
    card.appendChild(body);

    const footer = document.createElement('div');
    footer.className = 'fight-card-footer';
    footer.textContent = `${kills} kill${kills === 1 ? '' : 's'}`;
    card.appendChild(footer);

    card.title = `${f.label || 'Skirmish'}\n${formatTime(f.start_ms)} - ${formatTime(f.end_ms)} (${f.duration_ms / 1000}s)\n${kills} kill${kills === 1 ? '' : 's'}`;
    card.addEventListener('click', () => {
      document.querySelectorAll('.fight-card').forEach(c => c.classList.remove('active'));
      card.classList.add('active');
      showFight(f, idx, data.info);
    });
    timeline.appendChild(card);
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
  const kills = fight.kill_events ? fight.kill_events.length : 0;
  fightTitle.textContent = `Fight ${idx + 1} — ${fight.label || 'Skirmish'} @ ${formatTime(fight.start_ms)} - ${formatTime(fight.end_ms)} (${kills} kill${kills === 1 ? '' : 's'})`;
  fightMeter.innerHTML = '';

  const identities = {};
  if (info && info.info && info.info.participants) {
    info.info.participants.forEach(p => {
      identities[p.participantId] = p.championName || `P${p.participantId}`;
    });
  }

  const entries = fight.participants
    .map(pid => {
      const id = pid;
      const pidStr = String(pid);
      const name = identities[id] || `P${pidStr}`;
      const dmg = (fight.damage_by_participant[pidStr] || 0);
      const kills = (fight.kills_by_participant[pidStr] || 0);
      const assists = (fight.assists_by_participant[pidStr] || 0);
      const deaths = (fight.deaths_by_participant[pidStr] || 0);
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
