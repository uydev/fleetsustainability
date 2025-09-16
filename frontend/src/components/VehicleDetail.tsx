import React, { useEffect, useMemo, useState } from 'react';
import { Box, Paper, Typography, FormControl, InputLabel, Select, MenuItem, Button } from '@mui/material';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import apiService from '../services/api';
import { Telemetry, Vehicle } from '../types';

type Props = { vehicles: Vehicle[]; timeRange?: { from?: string; to?: string }; selectedVehicleId?: string | null };

const VehicleDetail: React.FC<Props> = ({ vehicles, timeRange, selectedVehicleId }) => {
  const [vehicleId, setVehicleId] = useState<string>(vehicles[0]?.id || '');
  const [data, setData] = useState<Telemetry[]>([]);
  const [rangeMs, setRangeMs] = useState<{ from: number; to: number } | null>(null);
  const [autoPickDone, setAutoPickDone] = useState<boolean>(false);

  const selectedVehicle = useMemo(() => vehicles.find(v => v.id === vehicleId), [vehicles, vehicleId]);
  const selectedType = selectedVehicle?.type;

  useEffect(() => {
    if (selectedVehicleId && vehicles.some(v => v.id === selectedVehicleId)) {
      setVehicleId(selectedVehicleId);
    } else if (!vehicleId && vehicles.length > 0) {
      setVehicleId(vehicles[0].id);
    }
  }, [vehicles, vehicleId, selectedVehicleId]);

  useEffect(() => {
    if (!vehicleId) return;
    // reset auto-pick when user changes range
    setAutoPickDone(false);
    let fromISO: string | undefined = timeRange?.from;
    let toISO: string | undefined = timeRange?.to;
    if (!fromISO && !toISO) {
      const to = new Date();
      const from = new Date(Date.now() - 24 * 3600 * 1000);
      fromISO = from.toISOString();
      toISO = to.toISOString();
    }
    apiService
      .getTelemetryByVehicle(vehicleId, { from: fromISO, to: toISO })
      .then(async (d) => {
        // If no data OR data is stale (older than 2 minutes), auto-follow the freshest vehicle in range
        const nowMs = Date.now();
        const stale = !d || d.length === 0 || (new Date(d[d.length - 1]?.timestamp || 0).getTime() < nowMs - 2 * 60 * 1000);
        if (stale && !autoPickDone) {
          let best: { v: Vehicle; rows: Telemetry[] } | null = null;
          for (const v of vehicles) {
            const fb = await apiService.getTelemetryByVehicle(v.id, { from: fromISO!, to: toISO! });
            if (fb && fb.length > 0) {
              if (!best || new Date(fb[fb.length - 1].timestamp).getTime() > new Date(best.rows[best.rows.length - 1].timestamp).getTime()) {
                best = { v, rows: fb };
              }
            }
          }
          if (best) {
            setVehicleId(best.v.id);
            setData(best.rows);
            setAutoPickDone(true);
            if (fromISO && toISO) setRangeMs({ from: new Date(fromISO).getTime(), to: new Date(toISO).getTime() });
            return;
          }
          // No data in this window for any vehicle: display empty but keep the requested window
          setData([]);
          if (fromISO && toISO) setRangeMs({ from: new Date(fromISO).getTime(), to: new Date(toISO).getTime() });
          setAutoPickDone(true);
          return;
        }
        setData(d);
        if (fromISO && toISO) {
          setRangeMs({ from: new Date(fromISO).getTime(), to: new Date(toISO).getTime() });
        }
      })
      .catch(() => setData([]));
  }, [vehicleId, timeRange?.from, timeRange?.to, vehicles, autoPickDone]);

  const spanMs = useMemo(() => (rangeMs ? Math.max(0, rangeMs.to - rangeMs.from) : undefined), [rangeMs]);

  const chartData = useMemo(() => {
    const sorted = (data || [])
      .slice()
      .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());

    // Collapse multiple samples within the same second (keep latest)
    const bySecond = new Map<number, typeof sorted[number]>();
    for (const t of sorted) {
      const ms = new Date(t.timestamp).getTime();
      const sec = Math.floor(ms / 1000);
      bySecond.set(sec, t);
    }

    const secs = Array.from(bySecond.keys()).sort((a, b) => a - b);
    const gapMs = 2000; // break lines if gap > 2s
    const rows: Array<{ tsMs: number; ts: string; speed: number | null; fuel?: number | null; battery?: number | null; emissions: number | null }> = [];
    let prevSec: number | undefined;

    for (const sec of secs) {
      const ms = sec * 1000;
      if (prevSec !== undefined && (sec - prevSec) * 1000 > gapMs) {
        rows.push({ tsMs: prevSec * 1000 + 1, ts: new Date(prevSec * 1000 + 1).toLocaleTimeString(), speed: null, fuel: null, battery: null, emissions: null });
      }
      const t = bySecond.get(sec)!;
      rows.push({
        tsMs: ms,
        ts: new Date(ms).toLocaleTimeString(),
        speed: Number((Math.round(t.speed * 10) / 10).toFixed(1)),
        fuel: t.fuel_level !== undefined ? Number((Math.round(t.fuel_level * 10) / 10).toFixed(1)) : undefined,
        battery: t.battery_level !== undefined ? Number((Math.round(t.battery_level * 10) / 10).toFixed(1)) : undefined,
        emissions: Number((Math.round(((selectedType === 'EV' ? 0 : t.emissions) * 10)) / 10).toFixed(1)),
      });
      prevSec = sec;
    }

    const filtered = rangeMs
      ? rows.filter((r) => typeof r.tsMs === 'number' && r.tsMs >= rangeMs.from && r.tsMs <= rangeMs.to)
      : rows;

    // Bucket/aggregate with higher resolution for short windows
    let bucketMs = 5000; // default 5s for very small ranges
    const span = spanMs ?? (filtered.length ? (filtered[filtered.length - 1].tsMs - filtered[0].tsMs) : 0);
    if (span <= 15 * 60 * 1000) {
      bucketMs = 5 * 1000; // ≤15m → 5s
    } else if (span <= 60 * 60 * 1000) {
      bucketMs = 30 * 1000; // ≤1h → 30s
    } else if (span <= 24 * 3600 * 1000) {
      bucketMs = 5 * 60 * 1000; // ≤24h → 5 min
    } else if (span <= 7 * 24 * 3600 * 1000) {
      bucketMs = 60 * 60 * 1000; // ≤7d → 1h
    } else {
      bucketMs = 3 * 60 * 60 * 1000; // >7d → 3h for more detail
    }

    const aggMap = new Map<number, { tsMs: number; speedSum: number; speedCount: number; speedMin: number; fuelSum: number; fuelCount: number; batterySum: number; batteryCount: number; emissionsSum: number; emissionsCount: number }>();
    for (const r of filtered) {
      const key = Math.floor(r.tsMs / bucketMs) * bucketMs;
      const cur = aggMap.get(key) || { tsMs: key, speedSum: 0, speedCount: 0, speedMin: Number.POSITIVE_INFINITY, fuelSum: 0, fuelCount: 0, batterySum: 0, batteryCount: 0, emissionsSum: 0, emissionsCount: 0 };
      if (typeof r.speed === 'number') { cur.speedSum += r.speed; cur.speedCount += 1; if (r.speed < cur.speedMin) cur.speedMin = r.speed; }
      if (typeof r.emissions === 'number') { cur.emissionsSum += r.emissions; cur.emissionsCount += 1; }
      if (typeof r.fuel === 'number') { cur.fuelSum += r.fuel; cur.fuelCount += 1; }
      if (typeof r.battery === 'number') { cur.batterySum += r.battery; cur.batteryCount += 1; }
      aggMap.set(key, cur);
    }

    let aggregated = Array.from(aggMap.values()).sort((a, b) => a.tsMs - b.tsMs).map((b) => ({
      tsMs: b.tsMs,
      ts: new Date(b.tsMs).toLocaleTimeString(),
      speed: b.speedCount ? Number((Math.round((b.speedSum / b.speedCount) * 10) / 10).toFixed(1)) : null,
      speedMin: (b.speedCount && b.speedMin !== Number.POSITIVE_INFINITY) ? Number(b.speedMin.toFixed(1)) : null,
      fuel: b.fuelCount ? Number((Math.round((b.fuelSum / b.fuelCount) * 10) / 10).toFixed(1)) : undefined,
      battery: b.batteryCount ? Number((Math.round((b.batterySum / b.batteryCount) * 10) / 10).toFixed(1)) : undefined,
      emissions: b.emissionsCount ? Number((Math.round(((b.emissionsSum / b.emissionsCount) * 10)) / 10).toFixed(1)) : null,
    }));

    // Forward-fill fuel/battery to show a continuous level curve without gaps (no backfill to avoid false flat lines)
    if (selectedType === 'EV') {
      let last: number | undefined = undefined;
      aggregated = aggregated.map((r) => {
        const val = r.battery !== undefined ? r.battery : last;
        last = val as number | undefined;
        return { ...r, battery: val };
      });
    } else if (selectedType === 'ICE') {
      let last: number | undefined = undefined;
      aggregated = aggregated.map((r) => {
        const val = r.fuel !== undefined ? r.fuel : last;
        last = val as number | undefined;
        return { ...r, fuel: val };
      });
    }

    return aggregated;
  }, [data, selectedType, rangeMs, spanMs]);

  const hasData = useMemo(() => chartData.some(d => d.speed !== null || d.emissions !== null || d.fuel !== undefined || d.battery !== undefined), [chartData]);

  const formatTick = (v: number) => {
    const d = new Date(v);
    if (!spanMs) return d.toLocaleTimeString();
    if (spanMs <= 2 * 3600 * 1000) {
      return d.toLocaleTimeString(undefined, { hour12: false, hour: '2-digit', minute: '2-digit' });
    }
    if (spanMs <= 48 * 3600 * 1000) {
      return d.toLocaleString(undefined, { month: '2-digit', day: '2-digit', hour: '2-digit' });
    }
    if (spanMs <= 14 * 24 * 3600 * 1000) {
      return d.toLocaleDateString(undefined, { month: '2-digit', day: '2-digit' });
    }
    return d.toLocaleDateString(undefined, { month: '2-digit', day: '2-digit' });
  };

  const exportCsv = () => {
    const headers = ['timestamp', 'speed', 'fuel_level', 'battery_level', 'emissions'];
    const rows = (data || []).map((d) => [d.timestamp, d.speed, d.fuel_level ?? '', d.battery_level ?? '', selectedType === 'EV' ? 0 : d.emissions]);
    const csv = [headers.join(','), ...rows.map((r) => r.join(','))].join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `vehicle_${vehicleId}_telemetry.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <Paper sx={{ p: 2 }}>
      <Box display="flex" alignItems="center" gap={2} mb={2}>
        <FormControl size="small">
          <InputLabel id="veh-label">Vehicle</InputLabel>
          <Select
            labelId="veh-label"
            label="Vehicle"
            value={vehicleId}
            onChange={(e) => setVehicleId(String(e.target.value))}
          >
            {vehicles.map((v) => (
              <MenuItem key={v.id} value={v.id}>
                {v.id} ({v.type})
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <Button variant="outlined" onClick={exportCsv}>
          Export CSV
        </Button>
      </Box>

      <Typography variant="subtitle1" gutterBottom>
        Speed & Emissions ({rangeMs ? `${new Date(rangeMs.from).toLocaleString()} → ${new Date(rangeMs.to).toLocaleString()}` : (timeRange?.from || timeRange?.to ? 'filtered' : 'last 24h')})
      </Typography>
      <Box sx={{ width: '100%', height: 280 }}>
        <ResponsiveContainer>
          <LineChart data={chartData} margin={{ top: 10, right: 20, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis
              dataKey="tsMs"
              type="number"
              domain={rangeMs ? [rangeMs.from, rangeMs.to] : ["auto", "auto"]}
              tickFormatter={formatTick}
            />
            <YAxis yAxisId="left" label={{ value: 'km/h', angle: -90, position: 'insideLeft' }} />
            <YAxis yAxisId="right" orientation="right" label={{ value: 'g/km', angle: 90, position: 'insideRight' }} />
            <Tooltip labelFormatter={(v) => formatTick(Number(v))} />
            <Legend />
            <Line yAxisId="left" type="monotone" dataKey="speed" stroke="#1976d2" dot={{ r: 2 }} connectNulls={false} name="Speed (avg)" />
            <Line yAxisId="left" type="monotone" dataKey="speedMin" stroke="#90caf9" strokeDasharray="4 4" dot={false} connectNulls={false} name="Speed (min)" />
            <Line yAxisId="right" type="monotone" dataKey="emissions" stroke="#ef6c00" dot={{ r: 2 }} connectNulls={false} name="Emissions" />
          </LineChart>
        </ResponsiveContainer>
      </Box>

      <Typography variant="subtitle1" gutterBottom mt={3}>
        Fuel/Battery (%)
      </Typography>
      <Box sx={{ width: '100%', height: 240 }}>
        <ResponsiveContainer>
          <LineChart data={chartData} margin={{ top: 10, right: 20, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis
              dataKey="tsMs"
              type="number"
              domain={rangeMs ? [rangeMs.from, rangeMs.to] : ["auto", "auto"]}
              tickFormatter={formatTick}
            />
            <YAxis label={{ value: '%', angle: -90, position: 'insideLeft' }} />
            <Tooltip labelFormatter={(v) => formatTick(Number(v))} />
            <Legend />
            {selectedType === 'ICE' && (
              <Line type="monotone" dataKey="fuel" stroke="#2e7d32" dot={{ r: 2 }} connectNulls={false} name="Fuel %" />
            )}
            {selectedType === 'EV' && (
              <Line type="monotone" dataKey="battery" stroke="#1565c0" dot={{ r: 2 }} connectNulls={false} name="Battery %" />
            )}
          </LineChart>
        </ResponsiveContainer>
      </Box>
    </Paper>
  );
};

export default VehicleDetail;


