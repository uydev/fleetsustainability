import React, { useMemo } from 'react';
import { Paper, Typography, Table, TableHead, TableRow, TableCell, TableBody } from '@mui/material';
import { Telemetry } from '../types';

type Props = { latest: Telemetry[] };

const Leaderboard: React.FC<Props> = ({ latest }) => {
  const rows = useMemo(() => {
    return (latest||[]).map(t => ({
      id: t.vehicle_id,
      efficiency: Number.isFinite(t.emissions / Math.max(t.speed, 0.1)) ? (t.emissions / Math.max(t.speed, 0.1)) : 0,
      speed: t.speed,
      emissions: t.emissions,
    })).sort((a,b) => a.efficiency - b.efficiency).slice(0, 10);
  }, [latest]);

  return (
    <Paper sx={{ p:2 }}>
      <Typography variant="h6" gutterBottom>Efficiency Leaderboard (best kg/100km)</Typography>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Vehicle</TableCell>
            <TableCell align="right">kg/100km</TableCell>
            <TableCell align="right">Speed (km/h)</TableCell>
            <TableCell align="right">Emissions (g/km)</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {rows.map(r => (
            <TableRow key={r.id}>
              <TableCell>{r.id}</TableCell>
              <TableCell align="right">{((Math.round((r.efficiency*100)*10))/10).toFixed(1)}</TableCell>
              <TableCell align="right">{(Math.round(r.speed*10)/10).toFixed(1)}</TableCell>
              <TableCell align="right">{(Math.round(r.emissions*10)/10).toFixed(1)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Paper>
  );
};

export default Leaderboard;


