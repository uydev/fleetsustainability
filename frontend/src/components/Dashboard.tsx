import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';
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
import WorldMap, { WorldMapRef } from './WorldMap';
import MetricsPanel from './MetricsPanel';
import VehicleDetail from './VehicleDetail';
import AdvancedMetrics from './AdvancedMetrics';
import Leaderboard from './Leaderboard';
import VehicleList from './VehicleList';
import TimeRangeSelector from './TimeRangeSelector';
import FleetManagement from './FleetManagement';
import TelemetryManagement from './TelemetryManagement';
import TripManagement from './TripManagement';
import MaintenanceManagement from './MaintenanceManagement';
import CostManagement from './CostManagement';
import LiveView from './LiveView';
import ElectrificationPlanning from './ElectrificationPlanning';
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
  const [currentTimeRange, setCurrentTimeRange] = useState<{ from?: string; to?: string }>({});
  const [alerts, setAlerts] = useState<Array<{type:string; vehicle_id:string; value:number; ts:string}>>([]);
  const [selectedVehicleId, setSelectedVehicleId] = useState<string | null>(null);
  
  // Reference to the WorldMap component
  const worldMapRef = useRef<WorldMapRef>(null);

  // Function to handle vehicle focus from vehicle list
  const handleVehicleFocus = (vehicleId: string) => {
    if (worldMapRef.current) {
      worldMapRef.current.focusOnVehicle(vehicleId);
    }
  };

  // Function to navigate to Vehicle Detail tab with specific vehicle ID
  const handleNavigateToVehicleDetail = (vehicleId: string) => {
    setSelectedVehicleId(vehicleId);
    setTabValue(2); // Vehicle Detail tab is at index 2
  };

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

  useEffect(() => {
    // Refresh alerts whenever time range changes
    const tr = currentTimeRange;
    apiService.getAlerts(tr && (tr.from || tr.to) ? tr : undefined).then(setAlerts).catch(() => setAlerts([]));
  }, [currentTimeRange.from, currentTimeRange.to]);

  // Build a telemetry array aligned 1:1 with registered vehicles
  // Uses the latest telemetry per vehicle when available, otherwise falls back to the vehicle's current_location
  const overviewTelemetry = useMemo(() => {
    // latest telemetry per vehicle_id
    const latestMap = new Map<string, Telemetry>();
    for (const t of telemetry) {
      const key = t.vehicle_id;
      const prev = latestMap.get(key);
      if (!prev || (t.timestamp && prev.timestamp && t.timestamp > prev.timestamp)) {
        latestMap.set(key, t);
      } else if (!prev) {
        latestMap.set(key, t);
      }
    }

    const mapped: Telemetry[] = [];
    for (const v of vehicles) {
      const lt = latestMap.get(v.id);
      if (lt) {
        mapped.push(lt);
        continue;
      }
      if (v.current_location && typeof v.current_location.lat === 'number' && typeof v.current_location.lon === 'number') {
        mapped.push({
          vehicle_id: v.id,
          timestamp: new Date().toISOString(),
          location: v.current_location,
          speed: 0,
          emissions: 0,
          type: v.type,
          status: v.status,
        } as Telemetry);
      }
    }
    return mapped;
  }, [telemetry, vehicles]);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
    // Clear selected vehicle ID when switching away from Vehicle Detail tab
    if (newValue !== 2) {
      setSelectedVehicleId(null);
    }
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
      <Container maxWidth="xl" sx={{ py: { xs: 2, sm: 3, md: 4 }, px: { xs: 1, sm: 2 } }}>

      <TimeRangeSelector onTimeRangeChange={handleTimeRangeChange} />

      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs 
          value={tabValue} 
          onChange={handleTabChange} 
          aria-label="dashboard tabs"
          variant="scrollable"
          scrollButtons="auto"
          allowScrollButtonsMobile
          sx={{
            '& .MuiTab-root': {
              minWidth: 'auto',
              px: 1.5,
              fontSize: '0.875rem',
            },
            '& .MuiTabs-scrollButtons': {
              '&.Mui-disabled': {
                opacity: 0.3,
              },
            },
          }}
        >
          <Tab label="Fleet Overview" />
          <Tab label="Live View" />
          <Tab label="Vehicle Detail" />
          <Tab label="Alerts" />
          <Tab label="Fleet Management" />
          <Tab label="Telemetry Management" />
          <Tab label="Trip Management" />
          <Tab label="Cost Management" />
          <Tab label="Maintenance" />
          <Tab label="Electrification" />
        </Tabs>
      </Box>

      <TabPanel value={tabValue} index={0}>
        <Grid container spacing={{ xs: 2, sm: 3 }}>
          <Grid item xs={12} lg={8}>
            <Paper sx={{ p: { xs: 1, sm: 2 }, height: { xs: '400px', sm: '500px', md: '100%' }, display: 'flex', flexDirection: 'column' }}>
              <Box sx={{ flex: 1, minHeight: 0 }}>
                <WorldMap ref={worldMapRef} telemetry={overviewTelemetry} />
              </Box>
            </Paper>
          </Grid>
          <Grid item xs={12} lg={4}>
            <Paper sx={{ p: { xs: 1, sm: 2 }, height: { xs: 'auto', lg: '100%' }, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
              <Box sx={{ flex: 1, overflow: 'auto' }}>
                <MetricsPanel metrics={metrics} />
                <Box mt={2}>
                  <AdvancedMetrics timeRange={currentTimeRange} />
                </Box>
              </Box>
            </Paper>
          </Grid>
          <Grid item xs={12}>
            <VehicleList telemetry={overviewTelemetry} onVehicleFocus={handleVehicleFocus} />
          </Grid>
          <Grid item xs={12}>
            <Leaderboard latest={overviewTelemetry} />
          </Grid>
        </Grid>
      </TabPanel>

        <TabPanel value={tabValue} index={1}>
          <LiveView onNavigateToVehicleDetail={handleNavigateToVehicleDetail} />
        </TabPanel>

      <TabPanel value={tabValue} index={2}>
        <VehicleDetail vehicles={vehicles} timeRange={currentTimeRange} selectedVehicleId={selectedVehicleId} />
      </TabPanel>

      <TabPanel value={tabValue} index={3}>
        <Paper sx={{ p:2 }}>
          <Typography variant="h6" gutterBottom>
            Alerts {currentTimeRange.from || currentTimeRange.to ? '(filtered)' : '(last hour)'}
          </Typography>
          {alerts.length === 0 ? (
            <Typography color="text.secondary">No alerts.</Typography>
          ) : (
            <Box display="flex" flexDirection="column" gap={1}>
              {alerts.map((a, i) => (
                <Box key={i} display="flex" justifyContent="space-between">
                  <Typography variant="body2">{a.type.replace('_',' ')} – Vehicle {a.vehicle_id}</Typography>
                  <Typography variant="body2">{new Date(a.ts).toLocaleString()}</Typography>
                </Box>
              ))}
            </Box>
          )}
        </Paper>
      </TabPanel>

      <TabPanel value={tabValue} index={4}>
        <FleetManagement timeRange={currentTimeRange} />
      </TabPanel>

      <TabPanel value={tabValue} index={5}>
        <TelemetryManagement timeRange={currentTimeRange} />
      </TabPanel>

      <TabPanel value={tabValue} index={6}>
        <TripManagement timeRange={currentTimeRange} />
      </TabPanel>

      <TabPanel value={tabValue} index={7}>
        <CostManagement timeRange={currentTimeRange} />
      </TabPanel>

      <TabPanel value={tabValue} index={8}>
        <MaintenanceManagement timeRange={currentTimeRange} />
      </TabPanel>

      <TabPanel value={tabValue} index={9}>
        <ElectrificationPlanning vehicles={vehicles} timeRange={currentTimeRange} />
      </TabPanel>
    </Container>
    </>
  );
};

export default Dashboard; 