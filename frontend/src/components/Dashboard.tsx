import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Grid,
  Paper,
  Typography,
  Container,
  Tabs,
  Tab,
  AppBar,
  Toolbar,
  Button,
  Avatar,
  Menu,
  MenuItem,
  IconButton,
} from '@mui/material';
import { AccountCircle, Logout } from '@mui/icons-material';
import FleetMap from './FleetMap';
import MetricsPanel from './MetricsPanel';
import VehicleList from './VehicleList';
import TimeRangeSelector from './TimeRangeSelector';
import FleetManagement from './FleetManagement';
import TelemetryManagement from './TelemetryManagement';
import TripManagement from './TripManagement';
import MaintenanceManagement from './MaintenanceManagement';
import CostManagement from './CostManagement';
import apiService from '../services/api';
import { useAuth } from '../contexts/AuthContext';
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
  const { user, logout } = useAuth();
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
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

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

  const [currentTimeRange, setCurrentTimeRange] = useState<{ from?: string; to?: string }>({});

  const handleTimeRangeChange = useCallback(async (timeRange: { from?: string; to?: string }) => {
    setCurrentTimeRange(timeRange);
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

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  const handleLogout = () => {
    logout();
    handleMenuClose();
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
    <>
      <AppBar position="static">
        <Toolbar>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            Fleet Sustainability Dashboard
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="body2" color="inherit">
              {user?.firstName} {user?.lastName} ({user?.role})
            </Typography>
            <IconButton
              size="large"
              aria-label="account of current user"
              aria-controls="menu-appbar"
              aria-haspopup="true"
              onClick={handleMenuOpen}
              color="inherit"
            >
              <AccountCircle />
            </IconButton>
            <Menu
              id="menu-appbar"
              anchorEl={anchorEl}
              anchorOrigin={{
                vertical: 'top',
                horizontal: 'right',
              }}
              keepMounted
              transformOrigin={{
                vertical: 'top',
                horizontal: 'right',
              }}
              open={Boolean(anchorEl)}
              onClose={handleMenuClose}
            >
              <MenuItem onClick={handleLogout}>
                <Logout sx={{ mr: 1 }} />
                Logout
              </MenuItem>
            </Menu>
          </Box>
        </Toolbar>
      </AppBar>
      <Container maxWidth="xl" sx={{ py: 4 }}>

      <TimeRangeSelector onTimeRangeChange={handleTimeRangeChange} />

      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs value={tabValue} onChange={handleTabChange} aria-label="dashboard tabs">
          <Tab label="Fleet Overview" />
          <Tab label="Fleet Management" />
          <Tab label="Telemetry Management" />
          <Tab label="Trip Management" />
          <Tab label="Maintenance" />
          <Tab label="Cost Management" />
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
        <FleetManagement timeRange={currentTimeRange} />
      </TabPanel>

      <TabPanel value={tabValue} index={2}>
        <TelemetryManagement timeRange={currentTimeRange} />
      </TabPanel>

      <TabPanel value={tabValue} index={3}>
        <TripManagement timeRange={currentTimeRange} />
      </TabPanel>

      <TabPanel value={tabValue} index={4}>
        <MaintenanceManagement timeRange={currentTimeRange} />
      </TabPanel>

      <TabPanel value={tabValue} index={5}>
        <CostManagement timeRange={currentTimeRange} />
      </TabPanel>
    </Container>
    </>
  );
};

export default Dashboard; 