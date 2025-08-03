import React from 'react';
import {
  Box,
  Typography,
  LinearProgress,
  Grid,
  Paper,
} from '@mui/material';
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  Tooltip,
} from 'recharts';
import { FleetMetrics } from '../types';

interface MetricsPanelProps {
  metrics: FleetMetrics;
}

const MetricsPanel: React.FC<MetricsPanelProps> = ({ metrics }) => {
  const evPercentage = metrics.ev_percent;
  const icePercentage = 100 - evPercentage;

  const pieData = [
    { name: 'Electric Vehicles', value: evPercentage, color: '#4caf50' },
    { name: 'ICE Vehicles', value: icePercentage, color: '#ff9800' },
  ];

  const formatEmissions = (emissions: number) => {
    if (emissions >= 1000) {
      return `${(emissions / 1000).toFixed(1)}k kg`;
    }
    return `${emissions.toFixed(1)} kg`;
  };

  const getEmissionsColor = (emissions: number) => {
    if (emissions < 100) return '#4caf50';
    if (emissions < 500) return '#ff9800';
    return '#f44336';
  };

  const getEvPercentageColor = (percentage: number) => {
    if (percentage >= 50) return '#4caf50';
    if (percentage >= 25) return '#ff9800';
    return '#f44336';
  };

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Grid container spacing={1} sx={{ flex: 1 }}>
        {/* Emissions Card */}
        <Grid item xs={12}>
          <Paper sx={{ p: 1.5 }}>
            <Typography variant="h6" gutterBottom>
              Total Emissions
            </Typography>
            <Typography variant="h4" color={getEmissionsColor(metrics.total_emissions)}>
              {formatEmissions(metrics.total_emissions)}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              CO₂ equivalent
            </Typography>
          </Paper>
        </Grid>

        {/* EV Percentage Card */}
        <Grid item xs={12}>
          <Paper sx={{ p: 1.5 }}>
            <Typography variant="h6" gutterBottom>
              Electric Vehicle Percentage
            </Typography>
            <Box display="flex" alignItems="center" mb={1}>
              <Typography variant="h4" color={getEvPercentageColor(evPercentage)}>
                {evPercentage.toFixed(1)}%
              </Typography>
            </Box>
            <LinearProgress
              variant="determinate"
              value={evPercentage}
              sx={{
                height: 8,
                borderRadius: 4,
                backgroundColor: '#e0e0e0',
                '& .MuiLinearProgress-bar': {
                  backgroundColor: getEvPercentageColor(evPercentage),
                },
              }}
            />
            <Typography variant="body2" color="text.secondary" mt={1}>
              Target: 50% by 2030
            </Typography>
          </Paper>
        </Grid>

        {/* Fleet Composition Chart */}
        <Grid item xs={12}>
          <Paper sx={{ p: 1.5, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
            <Typography variant="h6" gutterBottom align="center">
              Fleet Composition
            </Typography>
            <Box 
              sx={{ 
                width: '100%', 
                height: 150, 
                display: 'flex', 
                justifyContent: 'center', 
                alignItems: 'center' 
              }}
            >
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={pieData}
                    cx="50%"
                    cy="50%"
                    innerRadius={30}
                    outerRadius={60}
                    paddingAngle={5}
                    dataKey="value"
                  >
                    {pieData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    formatter={(value: number) => [`${value.toFixed(1)}%`, 'Percentage']}
                  />
                </PieChart>
              </ResponsiveContainer>
            </Box>
            <Box display="flex" justifyContent="space-around" mt={1} width="100%">
              <Box display="flex" alignItems="center">
                <Box
                  width={12}
                  height={12}
                  borderRadius="50%"
                  bgcolor="#4caf50"
                  mr={1}
                />
                <Typography variant="body2">EV: {evPercentage.toFixed(1)}%</Typography>
              </Box>
              <Box display="flex" alignItems="center">
                <Box
                  width={12}
                  height={12}
                  borderRadius="50%"
                  bgcolor="#ff9800"
                  mr={1}
                />
                <Typography variant="body2">ICE: {icePercentage.toFixed(1)}%</Typography>
              </Box>
            </Box>
          </Paper>
        </Grid>

        {/* Total Records */}
        <Grid item xs={12}>
          <Paper sx={{ p: 1.5 }}>
            <Typography variant="h6" gutterBottom>
              Data Points
            </Typography>
            <Typography variant="h4" color="primary">
              {metrics.total_records.toLocaleString()}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Telemetry records collected
            </Typography>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default MetricsPanel; 