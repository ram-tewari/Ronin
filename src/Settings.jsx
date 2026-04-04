import React, { useState, useEffect, useMemo } from 'react';

// Sport card config — emoji icons, colors, descriptions
const SPORT_META = {
  ncaa_mbb: {
    icon: '\u{1F3C0}',
    label: 'NCAA Basketball',
    desc: "Men's College Basketball",
    gradient: 'from-blue-600/20 to-blue-900/40',
    border: 'border-blue-500/50',
    accent: 'text-blue-400',
    glow: 'hover:shadow-blue-500/20',
    checkbox: 'text-blue-500',
    tag: 'bg-blue-900/40 text-blue-300',
    ring: 'ring-blue-500/40',
  },
  cricket_intl: {
    icon: '\u{1F3CF}',
    label: 'International Cricket',
    desc: 'Test, ODI & T20I Nations',
    gradient: 'from-emerald-600/20 to-emerald-900/40',
    border: 'border-emerald-500/50',
    accent: 'text-emerald-400',
    glow: 'hover:shadow-emerald-500/20',
    checkbox: 'text-emerald-500',
    tag: 'bg-emerald-900/40 text-emerald-300',
    ring: 'ring-emerald-500/40',
  },
  cricket_t20: {
    icon: '\u{26A1}',
    label: 'T20 Leagues',
    desc: 'IPL, BBL, PSL & more',
    gradient: 'from-amber-600/20 to-amber-900/40',
    border: 'border-amber-500/50',
    accent: 'text-amber-400',
    glow: 'hover:shadow-amber-500/20',
    checkbox: 'text-amber-500',
    tag: 'bg-amber-900/40 text-amber-300',
    ring: 'ring-amber-500/40',
  },
};

export default function Settings() {
  const [activeTab, setActiveTab] = useState('data');
  const [config, setConfig] = useState({
    data: { selectedTeams: [] },
    aesthetics: { themeColor: '#1a5fa8', bubbleDarkMode: false }
  });

  // Discovery state
  const [sports, setSports] = useState([]);
  const [discoveryLoading, setDiscoveryLoading] = useState(true);
  const [discoveryError, setDiscoveryError] = useState(null);

  // Drill-down: null = sport grid, sportId = team picker
  const [activeSport, setActiveSport] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');

  // Load saved config from Electron
  useEffect(() => {
    if (window.ronin?.getSettings) {
      window.ronin.getSettings().then((saved) => {
        if (saved) {
          if (saved.data && !Array.isArray(saved.data.selectedTeams)) {
            saved.data = { selectedTeams: [] };
          }
          setConfig(saved);
        }
      });
    }
  }, []);

  // Fetch discovery data from Go backend
  useEffect(() => {
    setDiscoveryLoading(true);
    setDiscoveryError(null);

    fetch('http://localhost:8080/discovery')
      .then(res => {
        if (!res.ok) throw new Error(`Discovery failed (${res.status})`);
        return res.json();
      })
      .then(data => {
        setSports(data.sports || []);
        setDiscoveryLoading(false);
      })
      .catch(err => {
        console.error('Discovery fetch failed:', err);
        setDiscoveryError(err.message);
        setDiscoveryLoading(false);
      });
  }, []);

  const selectedSet = useMemo(() => new Set(config.data.selectedTeams), [config.data.selectedTeams]);

  // The currently drilled-into sport object
  const currentSport = useMemo(() => {
    if (!activeSport) return null;
    return sports.find(s => s.id === activeSport) || null;
  }, [activeSport, sports]);

  // Filtered teams for the active sport
  const filteredTeams = useMemo(() => {
    if (!currentSport) return [];
    if (!searchQuery.trim()) return currentSport.teams;
    const q = searchQuery.toLowerCase();
    return currentSport.teams.filter(t => t.name.toLowerCase().includes(q));
  }, [currentSport, searchQuery]);

  const toggleTeam = (teamId) => {
    setConfig(prev => {
      const current = new Set(prev.data.selectedTeams);
      if (current.has(teamId)) {
        current.delete(teamId);
      } else {
        current.add(teamId);
      }
      return {
        ...prev,
        data: { ...prev.data, selectedTeams: Array.from(current) }
      };
    });
  };

  const toggleAllInSport = (sport, select) => {
    setConfig(prev => {
      const current = new Set(prev.data.selectedTeams);
      for (const team of sport.teams) {
        if (select) {
          current.add(team.id);
        } else {
          current.delete(team.id);
        }
      }
      return {
        ...prev,
        data: { ...prev.data, selectedTeams: Array.from(current) }
      };
    });
  };

  const updateAesthetics = (key, value) => {
    setConfig(prev => ({
      ...prev,
      aesthetics: { ...prev.aesthetics, [key]: value }
    }));
  };

  const handleSave = async () => {
    if (window.ronin?.saveSettings) {
      window.ronin.saveSettings(config);
    }

    try {
      await fetch('http://localhost:8080/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ selectedTeams: config.data.selectedTeams })
      });
    } catch (err) {
      console.error('Failed to sync config to backend:', err);
    }
  };

  const selectedCount = config.data.selectedTeams.length;

  // Count selected per sport
  const countForSport = (sport) => {
    if (!sport) return 0;
    return sport.teams.filter(t => selectedSet.has(t.id)).length;
  };

  return (
    <div className="w-full h-screen bg-neutral-900 text-neutral-100 flex flex-col font-sans">
      {/* Tabs */}
      <div className="flex px-4 pt-4 pb-2 border-b border-neutral-800 space-x-4">
        <button
          onClick={() => { setActiveTab('data'); setActiveSport(null); setSearchQuery(''); }}
          className={`pb-2 px-2 border-b-2 transition-colors ${activeTab === 'data' ? 'border-blue-500 text-blue-400' : 'border-transparent hover:text-neutral-300'}`}
        >
          Data Tracking
        </button>
        <button
          onClick={() => setActiveTab('aesthetics')}
          className={`pb-2 px-2 border-b-2 transition-colors ${activeTab === 'aesthetics' ? 'border-purple-500 text-purple-400' : 'border-transparent hover:text-neutral-300'}`}
        >
          Aesthetics
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 p-6 overflow-y-auto">
        {activeTab === 'data' && !activeSport && (
          <div className="space-y-5">
            <div className="flex items-center justify-between mb-1">
              <h2 className="text-xl font-bold">Track Teams</h2>
              <span className="text-sm text-neutral-400">{selectedCount} selected</span>
            </div>
            <p className="text-xs text-neutral-500 -mt-3">Choose a sport to browse and select teams.</p>

            {/* Loading / Error states */}
            {discoveryLoading && (
              <div className="text-center py-12 text-neutral-400">
                <div className="animate-pulse text-sm">Discovering available sports...</div>
              </div>
            )}

            {discoveryError && (
              <div className="p-4 bg-red-900/30 border border-red-800 rounded-lg text-red-300 text-sm">
                Failed to load teams: {discoveryError}
                <div className="mt-2 text-xs text-red-400">Make sure the Go backend is running on port 8080.</div>
              </div>
            )}

            {/* Sport Cards Grid */}
            {!discoveryLoading && !discoveryError && (
              <div className="grid grid-cols-1 gap-3">
                {sports.map(sport => {
                  const meta = SPORT_META[sport.id] || SPORT_META.ncaa_mbb;
                  const count = countForSport(sport);

                  return (
                    <button
                      key={sport.id}
                      onClick={() => { setActiveSport(sport.id); setSearchQuery(''); }}
                      className={`
                        relative w-full text-left p-4 rounded-xl border bg-gradient-to-br
                        ${meta.gradient} ${meta.border}
                        hover:shadow-lg ${meta.glow}
                        transition-all duration-200 group
                      `}
                    >
                      <div className="flex items-center gap-4">
                        {/* Icon */}
                        <div className="text-3xl flex-shrink-0 group-hover:scale-110 transition-transform duration-200">
                          {meta.icon}
                        </div>

                        {/* Info */}
                        <div className="flex-1 min-w-0">
                          <div className={`font-semibold text-sm ${meta.accent}`}>{meta.label}</div>
                          <div className="text-xs text-neutral-400 mt-0.5">{meta.desc}</div>
                        </div>

                        {/* Badge + Arrow */}
                        <div className="flex items-center gap-2 flex-shrink-0">
                          {count > 0 && (
                            <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${meta.tag}`}>
                              {count}
                            </span>
                          )}
                          <span className="text-neutral-500 group-hover:text-neutral-300 transition-colors text-sm">
                            &#x203A;
                          </span>
                        </div>
                      </div>

                      {/* Team count subtitle */}
                      <div className="text-[10px] text-neutral-500 mt-2 pl-12">
                        {sport.teams.length} teams available
                      </div>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {activeTab === 'data' && activeSport && currentSport && (
          <div className="space-y-4">
            {/* Back button + header */}
            {(() => {
              const meta = SPORT_META[activeSport] || SPORT_META.ncaa_mbb;
              const count = countForSport(currentSport);
              return (
                <>
                  <button
                    onClick={() => { setActiveSport(null); setSearchQuery(''); }}
                    className="flex items-center gap-1.5 text-sm text-neutral-400 hover:text-neutral-200 transition-colors -mb-1"
                  >
                    <span>&#x2039;</span> Back to sports
                  </button>

                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className="text-2xl">{meta.icon}</span>
                      <div>
                        <h2 className={`text-lg font-bold ${meta.accent}`}>{meta.label}</h2>
                        <p className="text-xs text-neutral-500">{meta.desc}</p>
                      </div>
                    </div>
                    <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${meta.tag}`}>
                      {count}/{currentSport.teams.length}
                    </span>
                  </div>

                  {/* Search + bulk actions */}
                  <div className="flex gap-2">
                    <input
                      type="text"
                      placeholder={`Search ${meta.label.toLowerCase()}...`}
                      value={searchQuery}
                      onChange={e => setSearchQuery(e.target.value)}
                      className={`flex-1 bg-neutral-800 border border-neutral-700 rounded-lg p-2.5 outline-none focus:ring-1 ${meta.ring} text-sm placeholder-neutral-500`}
                    />
                    <button
                      onClick={() => toggleAllInSport(currentSport, true)}
                      className={`text-xs px-3 py-2 rounded-lg bg-neutral-800 hover:bg-neutral-700 ${meta.accent} transition-colors border border-neutral-700`}
                    >
                      All
                    </button>
                    <button
                      onClick={() => toggleAllInSport(currentSport, false)}
                      className="text-xs px-3 py-2 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-400 transition-colors border border-neutral-700"
                    >
                      None
                    </button>
                  </div>

                  {/* Team list */}
                  <div className="bg-neutral-800 rounded-xl overflow-hidden divide-y divide-neutral-700/50">
                    {filteredTeams.map(team => (
                      <label
                        key={team.id}
                        className="flex items-center gap-3 cursor-pointer px-4 py-2.5 hover:bg-neutral-700/50 transition-colors"
                      >
                        <input
                          type="checkbox"
                          className={`form-checkbox ${meta.checkbox} h-4 w-4 bg-neutral-900 border-neutral-600 rounded`}
                          checked={selectedSet.has(team.id)}
                          onChange={() => toggleTeam(team.id)}
                        />
                        <span className="text-sm flex-1">{team.name}</span>
                        {selectedSet.has(team.id) && (
                          <span className={`text-[10px] ${meta.accent} opacity-60`}>tracking</span>
                        )}
                      </label>
                    ))}
                    {filteredTeams.length === 0 && (
                      <div className="text-center py-6 text-neutral-500 text-sm">
                        No teams match "{searchQuery}"
                      </div>
                    )}
                  </div>
                </>
              );
            })()}
          </div>
        )}

        {activeTab === 'aesthetics' && (
          <div className="space-y-6">
            <h2 className="text-xl font-bold mb-4">Appearance</h2>

            <div className="p-4 bg-neutral-800 rounded-lg">
              <label className="block text-sm font-medium mb-2">Theme Color</label>
              <select
                className="w-full bg-neutral-900 border border-neutral-700 rounded-md p-2 outline-none focus:border-purple-500"
                value={config.aesthetics.themeColor}
                onChange={e => updateAesthetics('themeColor', e.target.value)}
              >
                <option value="#1a5fa8">Blue (#1a5fa8)</option>
                <option value="#cc1800">Red (#cc1800)</option>
                <option value="#3a5a3a">Green (#3a5a3a)</option>
                <option value="#ffaa00">Orange (#ffaa00)</option>
                <option value="#8f2022">Crimson (#8f2022)</option>
              </select>
            </div>

            <label className="flex items-center justify-between p-4 bg-neutral-800 rounded-lg cursor-pointer hover:bg-neutral-700 transition-colors">
              <span className="font-medium">Force Bubble Dark Mode</span>
              <input
                type="checkbox"
                className="form-checkbox text-purple-500 h-5 w-5 bg-neutral-900 border-neutral-600 rounded"
                checked={config.aesthetics.bubbleDarkMode}
                onChange={e => updateAesthetics('bubbleDarkMode', e.target.checked)}
              />
            </label>
          </div>
        )}
      </div>

      {/* Save button */}
      <div className="p-4 border-t border-neutral-800 bg-neutral-900 flex justify-between items-center">
        {activeTab === 'data' && selectedCount > 0 && (
          <span className="text-xs text-neutral-500">{selectedCount} team(s) will be tracked</span>
        )}
        {(activeTab !== 'data' || selectedCount === 0) && <span />}
        <button
          onClick={handleSave}
          className="px-6 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-md font-medium transition-colors shadow-lg shadow-blue-900/20"
        >
          Save Changes
        </button>
      </div>
    </div>
  );
}
