import React, { useEffect, useState } from 'react';
import { Box, Grid, Paper, Typography } from '@mui/material';
import apiService from '../services/api';

type Props = { timeRange?: { from?: string; to?: string } };

const fmt1 = (n: number | undefined) => {
  const v = Number(n);
  if (!isFinite(v)) return '0.0';
  return (Math.round(v * 10) / 10).toFixed(1);
};

// (fmt0 removed – unused)

const AdvancedMetrics: React.FC<Props> = ({ timeRange }) => {
  const [data, setData] = useState<any>({});
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    apiService.getAdvancedMetrics(timeRange)
      .then(setData)
      .catch(() => setData({}))
      .finally(() => setLoading(false));
  }, [timeRange?.from, timeRange?.to]);

  return (
    <Paper sx={{ p:2 }}>
      <Typography variant="h6" gutterBottom>Advanced Metrics</Typography>
      {loading ? (
        <Typography>Loading…</Typography>
      ) : (
        <Grid container spacing={2}>
          <Grid item xs={12} sm={6} md={3}>
            <Box>
              <Typography variant="body2" color="text.secondary">Fuel Used (pct)</Typography>
              <Typography variant="h5">{fmt1(data.fuel_used_pct)}</Typography>
            </Box>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Box>
              <Typography variant="body2" color="text.secondary">Energy Used (pct)</Typography>
              <Typography variant="h5">{fmt1(data.energy_used_pct)}</Typography>
            </Box>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Box>
              <Typography variant="body2" color="text.secondary">Cost Estimate</Typography>
              <Typography variant="h5">${fmt1(data.cost_estimate)}</Typography>
            </Box>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Box>
              <Typography variant="body2" color="text.secondary">Emissions</Typography>
              <Typography variant="h5">{fmt1(data.emissions)} kg</Typography>
            </Box>
          </Grid>
        </Grid>
      )}
      <Box mt={2}>
        <Typography variant="subtitle1">Electrification hint</Typography>
        <Typography variant="body2" color="text.secondary">
          Suggest replacing ICE vehicles with consistently high emissions and low efficiency. Use the leaderboard to identify candidates.
        </Typography>
      </Box>
    </Paper>
  );
};

export default AdvancedMetrics;


