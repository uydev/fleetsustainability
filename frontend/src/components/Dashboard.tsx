import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Grid,
  Paper,
  Typography,
  Card,
  CardContent,
  CircularProgress,
  Alert,
  Container,
} from '@mui/material';
import {
  DirectionsCar as CarIcon,
  BatteryChargingFull as EvIcon,
  LocalGasStation as GasIcon,
  TrendingUp as EmissionsIcon,
} from '@mui/icons-material';
import { Telemetry, FleetMetrics } from '../types';
import apiService from '../services/api';
import FleetMap from './FleetMap';
import MetricsPanel from './MetricsPanel';
import VehicleList from './VehicleList';
import TimeRangeSelector from './TimeRangeSelector';

const Dashboard: React.FC = () => {
  const [telemetry, setTelemetry] = useState<Telemetry[]>([]);
  const [metrics, setMetrics] = useState<FleetMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [timeRange, setTimeRange] = useState<{ from?: string; to?: string }>({});

  const loadDashboardData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const [telemetryData, metricsData] = await Promise.all([
        apiService.getTelemetry(timeRange),
        apiService.getFleetMetrics(timeRange),
      ]);

      setTelemetry(telemetryData);
      setMetrics(metricsData);
    } catch (err) {
      setError('Failed to load dashboard data. Please try again.');
      console.error('Dashboard load error:', err);
    } finally {
      setLoading(false);
    }
  }, [timeRange]);

  useEffect(() => {
    loadDashboardData();
  }, [loadDashboardData]);

  const handleTimeRangeChange = (newTimeRange: { from?: string; to?: string }) => {
    setTimeRange(newTimeRange);
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="60vh">
        <CircularProgress size={60} />
      </Box>
    );
  }

  if (error) {
    return (
      <Container maxWidth="lg" sx={{ mt: 4 }}>
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      </Container>
    );
  }

  return (
    <Container maxWidth="xl" sx={{ mt: 4, mb: 4 }}>
      <Typography variant="h3" component="h1" gutterBottom>
        Fleet Sustainability Dashboard
      </Typography>

      <TimeRangeSelector onTimeRangeChange={handleTimeRangeChange} />

      <Grid container spacing={3}>
        {/* Fleet Map */}
        <Grid item xs={12} lg={8}>
          <Paper sx={{ p: 2, height: 400 }}>
            <Typography variant="h6" gutterBottom>
              Fleet Overview
            </Typography>
            <FleetMap telemetry={telemetry} />
          </Paper>
        </Grid>

        {/* Metrics Panel */}
        <Grid item xs={12} lg={4}>
          <Paper sx={{ p: 2, height: 400 }}>
            <Typography variant="h6" gutterBottom>
              Fleet Metrics
            </Typography>
            {metrics && <MetricsPanel metrics={metrics} />}
          </Paper>
        </Grid>

        {/* Quick Stats Cards */}
        <Grid item xs={12}>
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6} md={3}>
              <Card>
                <CardContent>
                  <Box display="flex" alignItems="center">
                    <CarIcon color="primary" sx={{ mr: 1 }} />
                    <Box>
                      <Typography variant="h6">
                        {telemetry?.length || 0}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        Total Vehicles
                      </Typography>
                    </Box>
                  </Box>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card>
                <CardContent>
                  <Box display="flex" alignItems="center">
                    <EvIcon color="success" sx={{ mr: 1 }} />
                    <Box>
                      <Typography variant="h6">
                        {telemetry?.filter(t => t.type === 'EV').length || 0}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        Electric Vehicles
                      </Typography>
                    </Box>
                  </Box>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card>
                <CardContent>
                  <Box display="flex" alignItems="center">
                    <GasIcon color="warning" sx={{ mr: 1 }} />
                    <Box>
                      <Typography variant="h6">
                        {telemetry?.filter(t => t.type === 'ICE').length || 0}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        ICE Vehicles
                      </Typography>
                    </Box>
                  </Box>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card>
                <CardContent>
                  <Box display="flex" alignItems="center">
                    <EmissionsIcon color="error" sx={{ mr: 1 }} />
                    <Box>
                      <Typography variant="h6">
                        {metrics?.total_emissions.toFixed(1) || '0'}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        Total Emissions (kg)
                      </Typography>
                    </Box>
                  </Box>
                </CardContent>
              </Card>
            </Grid>
          </Grid>
        </Grid>

        {/* Vehicle List */}
        <Grid item xs={12}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>
              Vehicle Details
            </Typography>
            <VehicleList telemetry={telemetry} />
          </Paper>
        </Grid>
      </Grid>
    </Container>
  );
};

export default Dashboard; 