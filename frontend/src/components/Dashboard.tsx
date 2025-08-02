import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Grid,
  Paper,
  Typography,
  Container,
  Tabs,
  Tab,
} from '@mui/material';
import FleetMap from './FleetMap';
import MetricsPanel from './MetricsPanel';
import VehicleList from './VehicleList';
import TimeRangeSelector from './TimeRangeSelector';
import FleetManagement from './FleetManagement';
import apiService from '../services/api';
import { Telemetry, FleetMetrics, Vehicle } from '../types';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`simple-tabpanel-${index}`}
      aria-labelledby={`simple-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

const Dashboard: React.FC = () => {
  const [telemetry, setTelemetry] = useState<Telemetry[]>([]);
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [metrics, setMetrics] = useState<FleetMetrics>({
    total_emissions: 0,
    ev_percent: 0,
    total_records: 0,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tabValue, setTabValue] = useState(0);

  const loadDashboardData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [telemetryData, metricsData, vehiclesData] = await Promise.all([
        apiService.getTelemetry(),
        apiService.getFleetMetrics(),
        apiService.getVehicles(),
      ]);
      setTelemetry(telemetryData);
      setMetrics(metricsData);
      setVehicles(vehiclesData);
    } catch (err) {
      console.error('Error loading dashboard data:', err);
      setError('Failed to load dashboard data. Please try again.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadDashboardData();
  }, [loadDashboardData]);

  const handleTimeRangeChange = useCallback(async (timeRange: { from?: string; to?: string }) => {
    setLoading(true);
    try {
      const [telemetryData, metricsData] = await Promise.all([
        apiService.getTelemetry(timeRange),
        apiService.getFleetMetrics(timeRange),
      ]);
      setTelemetry(telemetryData);
      setMetrics(metricsData);
    } catch (err) {
      console.error('Error loading filtered data:', err);
      setError('Failed to load filtered data. Please try again.');
    } finally {
      setLoading(false);
    }
  }, []);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  if (loading && telemetry.length === 0) {
    return (
      <Container maxWidth="xl">
        <Typography>Loading dashboard...</Typography>
      </Container>
    );
  }

  if (error) {
    return (
      <Container maxWidth="xl">
        <Typography color="error">{error}</Typography>
      </Container>
    );
  }

  const evCount = telemetry?.filter(t => t.battery_level !== undefined && t.battery_level !== null).length || 0;
  const iceCount = telemetry?.filter(t => t.fuel_level !== undefined && t.fuel_level !== null).length || 0;

  return (
    <Container maxWidth="xl" sx={{ py: 4 }}>
      <Typography variant="h4" component="h1" gutterBottom>
        Fleet Sustainability Dashboard
      </Typography>

      <TimeRangeSelector onTimeRangeChange={handleTimeRangeChange} />

      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs value={tabValue} onChange={handleTabChange} aria-label="dashboard tabs">
          <Tab label="Fleet Overview" />
          <Tab label="Fleet Management" />
        </Tabs>
      </Box>

      <TabPanel value={tabValue} index={0}>
        <Grid container spacing={3}>
          <Grid item xs={12} lg={8}>
            <Paper sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column' }}>
              <Box sx={{ flex: 1, minHeight: 0 }}>
                <FleetMap telemetry={telemetry} />
              </Box>
            </Paper>
          </Grid>
          <Grid item xs={12} lg={4}>
            <Paper sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
              <Box sx={{ flex: 1, overflow: 'auto' }}>
                <MetricsPanel metrics={metrics} />
              </Box>
            </Paper>
          </Grid>
          <Grid item xs={12}>
            <VehicleList telemetry={telemetry} />
          </Grid>
        </Grid>
      </TabPanel>

      <TabPanel value={tabValue} index={1}>
        <FleetManagement />
      </TabPanel>
    </Container>
  );
};

export default Dashboard; 