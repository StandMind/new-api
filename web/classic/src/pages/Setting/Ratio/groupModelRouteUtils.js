/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export function parseGroupModelRouteGroups(groupRatio) {
  try {
    const parsed = JSON.parse(groupRatio || '{}');
    return Object.keys(parsed).sort();
  } catch {
    return [];
  }
}

export function getGroupModelRouteKey(group, model) {
  return JSON.stringify([group, model]);
}

export function cloneGroupModelRouteTiers(tiers) {
  return (tiers || []).map((tier) => ({
    editorId: crypto.randomUUID(),
    priority: tier.priority,
    channels: (tier.channels || []).map((channel) => ({ ...channel })),
  }));
}

export function serializeGroupModelRouteTiers(tiers) {
  return JSON.stringify(
    tiers.map((tier) => ({
      priority: tier.priority,
      channels: tier.channels,
    })),
  );
}

export function countGroupModelRouteChannels(route) {
  return route.tiers.reduce((total, tier) => total + tier.channels.length, 0);
}

export function groupModelRouteMatchesSearch(route, search) {
  const normalizedSearch = search.trim().toLowerCase();
  if (!normalizedSearch) return true;

  if (
    route.group.toLowerCase().includes(normalizedSearch) ||
    route.model.toLowerCase().includes(normalizedSearch)
  ) {
    return true;
  }

  return route.tiers.some((tier) =>
    tier.channels.some(
      (channel) =>
        String(channel.channel_id).includes(normalizedSearch) ||
        channel.channel_name.toLowerCase().includes(normalizedSearch),
    ),
  );
}
