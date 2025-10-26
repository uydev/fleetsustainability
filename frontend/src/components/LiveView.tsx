import React, { useEffect, useRef, useState } from 'react';
import WorldMap from './WorldMap';
import { Telemetry } from '../types';
import { Box, Typography, Paper } from '@mui/material';
import apiService from '../services/api';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8081';

// Simple in-memory position store keyed by vehicle_id
type TelemetryById = Map<string, Telemetry>;

type VehicleMeta = { id: string; type?: string; status?: string; current_location?: { lat: number; lon: number } };

interface LiveViewProps {
  onNavigateToVehicleDetail?: (vehicleId: string) => void;
}

const LiveView: React.FC<LiveViewProps> = ({ onNavigateToVehicleDetail }) => {
  const [telemetryList, setTelemetryList] = useState<Telemetry[]>([]);
  const [vehicles, setVehicles] = useState<VehicleMeta[]>([]);
  const storeRef = useRef<TelemetryById>(new Map());
  const eventSourceRef = useRef<EventSource | null>(null);
  const pollerRef = useRef<NodeJS.Timer | null>(null);

  // Compose exactly one item per vehicle id from union(registered vehicles, SSE vehicles)
  const currentTelemetry: Telemetry[] = (() => {
    const ids = new Set<string>();
    const out: Telemetry[] = [];

    for (const v of vehicles) {
      ids.add(v.id);
      const t = storeRef.current.get(v.id);
      if (t) {
        out.push({ ...t, vehicle_id: v.id, type: t.type || (v.type as any) || 'ICE', status: t.status || (v.status as any) || 'active' });
      } else {
        out.push({
          vehicle_id: v.id,
          timestamp: new Date().toISOString(),
          location: v.current_location || { lat: 0, lon: 0 },
          speed: 0,
          emissions: 0,
          type: (v.type as any) || 'ICE',
          status: (v.status as any) || 'active',
        } as Telemetry);
      }
    }

    storeRef.current.forEach((t, id) => {
      if (!ids.has(id)) {
        ids.add(id);
        out.push(t);
      }
    });

    return out;
  })();

  useEffect(() => {
    // Load registered vehicles once
    (async () => {
      try {
        const list = await apiService.getVehicles();
        setVehicles(Array.isArray(list) ? list : []);
      } catch {
        setVehicles([]);
      }
    })();

    // SSE stream (best-effort)
    const url = `${API_BASE_URL}/api/telemetry/stream`;
    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onmessage = (evt) => {
      try {
        const data = JSON.parse(evt.data);
        const rec: Telemetry = {
          vehicle_id: String(data.vehicle_id || ''),
          timestamp: String(data.timestamp || new Date().toISOString()),
          location: { lat: Number(data.location?.lat ?? 0), lon: Number(data.location?.lon ?? 0) },
          speed: Number(data.speed ?? 0),
          fuel_level: data.fuel_level !== undefined ? Number(data.fuel_level) : undefined,
          battery_level: data.battery_level !== undefined ? Number(data.battery_level) : undefined,
          emissions: Number(data.emissions ?? 0),
          type: data.type === 'EV' ? 'EV' : 'ICE',
          status: data.status === 'inactive' ? 'inactive' : 'active',
        };
        if (!rec.vehicle_id) return;
        storeRef.current.set(rec.vehicle_id, rec);
        // Trigger a re-render
        setTelemetryList((prev) => (prev.length > 200 ? [] : [...prev, rec]));
      } catch {}
    };

    // Polling fallback (also runs alongside SSE as backup)
    const startPolling = () => {
      if (pollerRef.current) return;
      pollerRef.current = setInterval(async () => {
        try {
          const from = new Date(Date.now() - 60 * 1000).toISOString();
          const list = await apiService.getTelemetry({ from });
          if (Array.isArray(list) && list.length) {
            const latestById = new Map<string, Telemetry>();
            for (const item of list) {
              if (!item?.vehicle_id) continue;
              const prev = latestById.get(item.vehicle_id);
              if (!prev || new Date(item.timestamp).getTime() > new Date(prev.timestamp).getTime()) {
                latestById.set(item.vehicle_id, item as Telemetry);
              }
            }
            let updated = false;
            latestById.forEach((t, id) => {
              const prev = storeRef.current.get(id);
              if (!prev || prev.timestamp !== t.timestamp) {
                storeRef.current.set(id, t);
                updated = true;
              }
            });
            if (updated) setTelemetryList((prev) => (prev.length > 200 ? [] : [...prev]));
          }
        } catch {}
      }, 2000);
    };

    es.onerror = () => {
      // Start polling if SSE isn't working
      startPolling();
    };

    // Always start polling as a safety net
    startPolling();

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (pollerRef.current) {
        clearInterval(pollerRef.current);
        pollerRef.current = null;
      }
    };
  }, []);

  return (
    <Box>
      <Paper sx={{ p: 2, mb: 2 }}>
        <Typography variant="h6">Live Fleet View</Typography>
        <Typography variant="body2" color="text.secondary">
          Real-time stream with automatic API polling fallback. Start the simulator to see movement.
        </Typography>
      </Paper>
      <WorldMap telemetry={currentTelemetry} onNavigateToVehicleDetail={onNavigateToVehicleDetail} />
    </Box>
  );
};

export default LiveView;


