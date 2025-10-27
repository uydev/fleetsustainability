import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Paper,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Avatar,
  Chip,
  LinearProgress,
  Grid,
  Card,
  CardContent,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Tooltip,
  Alert,
} from '@mui/material';
import {
  EmojiEvents,
  TrendingUp,
  TrendingDown,
  LocalGasStation,
  Park,
} from '@mui/icons-material';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import apiService from '../services/api';
import { DriverEfficiency, DriverLeaderboard, TimeRange, Telemetry, Vehicle } from '../types';

interface Props {
  timeRange?: TimeRange;
  vehicles?: Vehicle[];
  telemetry?: Telemetry[];
}

const DriverEfficiencyLeaderboard: React.FC<Props> = ({ timeRange, vehicles = [], telemetry = [] }) => {
  const [leaderboard, setLeaderboard] = useState<DriverLeaderboard | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedPeriod, setSelectedPeriod] = useState<'daily' | 'weekly' | 'monthly' | 'yearly'>('monthly');

  // Helper functions for calculations
  const calculateDistance = (loc1: { lat: number; lon: number }, loc2: { lat: number; lon: number }): number => {
    const R = 6371; // Earth's radius in km
    const dLat = (loc2.lat - loc1.lat) * Math.PI / 180;
    const dLon = (loc2.lon - loc1.lon) * Math.PI / 180;
    const a = Math.sin(dLat/2) * Math.sin(dLat/2) +
              Math.cos(loc1.lat * Math.PI / 180) * Math.cos(loc2.lat * Math.PI / 180) *
              Math.sin(dLon/2) * Math.sin(dLon/2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1-a));
    return R * c;
  };

  const calculateSpeedVariance = (telemetry: Telemetry[]): number => {
    if (telemetry.length < 2) return 0;
    const speeds = telemetry.map(t => t.speed);
    const mean = speeds.reduce((sum, speed) => sum + speed, 0) / speeds.length;
    const variance = speeds.reduce((sum, speed) => sum + Math.pow(speed - mean, 2), 0) / speeds.length;
    return Math.sqrt(variance);
  };

  const countTrips = (telemetry: Telemetry[]): number => {
    if (telemetry.length < 2) return 0;
    let trips = 0;
    let wasMoving = false;
    
    for (const t of telemetry) {
      const isMoving = t.speed > 5; // Consider moving if speed > 5 km/h
      if (isMoving && !wasMoving) {
        trips++;
      }
      wasMoving = isMoving;
    }
    return trips;
  };

  const calculateDrivingTime = (telemetry: Telemetry[]): number => {
    if (telemetry.length < 2) return 0;
    
    let drivingTime = 0;
    for (let i = 1; i < telemetry.length; i++) {
      const prev = telemetry[i - 1];
      const curr = telemetry[i];
      const timeDiff = (new Date(curr.timestamp).getTime() - new Date(prev.timestamp).getTime()) / 1000 / 3600; // hours
      
      if (curr.speed > 5) { // Only count time when actually moving
        drivingTime += timeDiff;
      }
    }
    return drivingTime;
  };

  const generateBadges = (efficiency: number, fuelEfficiency: number, emissionEfficiency: number, trips: number, safety: number): string[] => {
    const badges: string[] = [];
    
    if (efficiency >= 90) badges.push('Efficiency Master');
    if (fuelEfficiency >= 12) badges.push('Fuel Saver');
    if (emissionEfficiency <= 100) badges.push('Eco Champion');
    if (safety >= 85) badges.push('Safety Star');
    if (trips >= 10) badges.push('Road Warrior');
    if (efficiency >= 80) badges.push('Top Performer');
    
    return badges;
  };

  // Calculate driver efficiency from telemetry data
  const calculateDriverEfficiency = useMemo(() => {
    if (!vehicles.length || !telemetry.length) return null;

    // Group telemetry by vehicle
    const telemetryByVehicle = telemetry.reduce((acc, t) => {
      if (!acc[t.vehicle_id]) acc[t.vehicle_id] = [];
      acc[t.vehicle_id].push(t);
      return acc;
    }, {} as Record<string, Telemetry[]>);

    // Calculate efficiency for each vehicle (treating each vehicle as a "driver")
    const driverEfficiencies: DriverEfficiency[] = vehicles.map((vehicle, index) => {
      const vehicleTelemetry = telemetryByVehicle[vehicle.id] || [];
      
      if (vehicleTelemetry.length === 0) {
        return {
          driver_id: `driver_${vehicle.id}`,
          driver_name: `Driver ${index + 1}`,
          total_distance: 0,
          total_fuel_consumed: 0,
          total_emissions: 0,
          average_speed: 0,
          efficiency_score: 0,
          fuel_efficiency: 0,
          emission_efficiency: 0,
          safety_score: 0,
          trips_completed: 0,
          total_driving_time: 0,
          rank: vehicles.length - index,
          improvement_since_last_period: 0,
          badges: [],
        };
      }

      // Sort telemetry by timestamp
      const sortedTelemetry = vehicleTelemetry.sort((a, b) => 
        new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
      );

      // Calculate distance traveled
      let totalDistance = 0;
      for (let i = 1; i < sortedTelemetry.length; i++) {
        const prev = sortedTelemetry[i - 1];
        const curr = sortedTelemetry[i];
        const distance = calculateDistance(prev.location, curr.location);
        totalDistance += distance;
      }

      // Calculate fuel consumption
      const totalFuelConsumed = vehicle.type === 'ICE' 
        ? sortedTelemetry.reduce((sum, t) => sum + (t.fuel_level || 0), 0) / 100
        : sortedTelemetry.reduce((sum, t) => sum + (t.battery_level || 0), 0) / 100;

      // Calculate emissions
      const totalEmissions = sortedTelemetry.reduce((sum, t) => sum + t.emissions, 0);

      // Calculate average speed
      const averageSpeed = sortedTelemetry.reduce((sum, t) => sum + t.speed, 0) / sortedTelemetry.length;

      // Calculate fuel efficiency (km per unit of fuel/battery)
      const fuelEfficiency = totalDistance > 0 && totalFuelConsumed > 0 
        ? totalDistance / totalFuelConsumed 
        : 0;

      // Calculate emission efficiency (g CO2 per km)
      const emissionEfficiency = totalDistance > 0 ? totalEmissions / totalDistance : 0;

      // Calculate efficiency score (0-100)
      const speedScore = Math.min(100, (averageSpeed / 50) * 100); // Optimal around 50 km/h
      const fuelScore = vehicle.type === 'ICE' 
        ? Math.min(100, (fuelEfficiency / 15) * 100) // Optimal around 15 km/L
        : Math.min(100, (fuelEfficiency / 8) * 100); // Optimal around 8 km/kWh
      const emissionScore = Math.max(0, 100 - (emissionEfficiency / 200) * 100); // Lower is better
      const distanceScore = Math.min(100, (totalDistance / 1000) * 10); // Reward more distance
      
      const efficiencyScore = Math.round((speedScore * 0.2 + fuelScore * 0.3 + emissionScore * 0.3 + distanceScore * 0.2));

      // Calculate safety score based on speed consistency
      const speedVariance = calculateSpeedVariance(sortedTelemetry);
      const safetyScore = Math.max(0, 100 - speedVariance * 2);

      // Count trips (segments of continuous movement)
      const tripsCompleted = countTrips(sortedTelemetry);

      // Calculate total driving time
      const totalDrivingTime = calculateDrivingTime(sortedTelemetry);

      // Generate badges
      const badges = generateBadges(efficiencyScore, fuelEfficiency, emissionEfficiency, tripsCompleted, safetyScore);

      return {
        driver_id: `driver_${vehicle.id}`,
        driver_name: `Driver ${index + 1}`,
        total_distance: Math.round(totalDistance * 100) / 100,
        total_fuel_consumed: Math.round(totalFuelConsumed * 100) / 100,
        total_emissions: Math.round(totalEmissions * 100) / 100,
        average_speed: Math.round(averageSpeed * 100) / 100,
        efficiency_score: efficiencyScore,
        fuel_efficiency: Math.round(fuelEfficiency * 100) / 100,
        emission_efficiency: Math.round(emissionEfficiency * 100) / 100,
        safety_score: Math.round(safetyScore),
        trips_completed: tripsCompleted,
        total_driving_time: Math.round(totalDrivingTime * 100) / 100,
        rank: 0, // Will be set after sorting
        improvement_since_last_period: 0, // Would need historical data
        badges,
      };
    });

    // Sort by efficiency score and assign ranks
    driverEfficiencies.sort((a, b) => b.efficiency_score - a.efficiency_score);
    driverEfficiencies.forEach((driver, index) => {
      driver.rank = index + 1;
    });

    return {
      drivers: driverEfficiencies,
      period: selectedPeriod,
      total_drivers: driverEfficiencies.length,
      last_updated: new Date().toISOString(),
    };
  }, [vehicles, telemetry, selectedPeriod]);

  // Update leaderboard when calculations change
  useEffect(() => {
    if (calculateDriverEfficiency) {
      setLeaderboard(calculateDriverEfficiency);
      setLoading(false);
      setError(null);
    } else if (vehicles.length === 0 || telemetry.length === 0) {
      setLoading(false);
      setError('No vehicle or telemetry data available');
    }
  }, [calculateDriverEfficiency, vehicles.length, telemetry.length]);

  // Calculate summary statistics
  const summary = useMemo(() => {
    if (!leaderboard?.drivers.length) return null;

    const drivers = leaderboard.drivers;
    const totalDrivers = drivers.length;
    const avgEfficiency = drivers.reduce((sum: number, d) => sum + d.efficiency_score, 0) / totalDrivers;
    const avgFuelEfficiency = drivers.reduce((sum: number, d) => sum + d.fuel_efficiency, 0) / totalDrivers;
    const totalDistance = drivers.reduce((sum: number, d) => sum + d.total_distance, 0);
    const totalEmissions = drivers.reduce((sum: number, d) => sum + d.total_emissions, 0);

    return {
      totalDrivers,
      avgEfficiency: Math.round(avgEfficiency),
      avgFuelEfficiency: Math.round(avgFuelEfficiency * 10) / 10,
      totalDistance: Math.round(totalDistance),
      totalEmissions: Math.round(totalEmissions),
    };
  }, [leaderboard]);

  // Chart data for efficiency distribution
  const efficiencyChartData = useMemo(() => {
    if (!leaderboard?.drivers.length) return [];

    const ranges = [
      { range: '90-100', min: 90, max: 100, count: 0, color: '#4caf50' },
      { range: '80-89', min: 80, max: 89, count: 0, color: '#8bc34a' },
      { range: '70-79', min: 70, max: 79, count: 0, color: '#ffc107' },
      { range: '60-69', min: 60, max: 69, count: 0, color: '#ff9800' },
      { range: '0-59', min: 0, max: 59, count: 0, color: '#f44336' },
    ];

    leaderboard.drivers.forEach((driver) => {
      const score = driver.efficiency_score;
      const range = ranges.find(r => score >= r.min && score <= r.max);
      if (range) range.count++;
    });

    return ranges.filter(r => r.count > 0);
  }, [leaderboard]);

  // Badge system
  const getBadges = (driver: DriverEfficiency) => {
    const badges: string[] = [];
    
    if (driver.efficiency_score >= 95) badges.push('Efficiency Master');
    if (driver.fuel_efficiency >= 15) badges.push('Fuel Saver');
    if (driver.emission_efficiency <= 50) badges.push('Eco Champion');
    if (driver.safety_score >= 90) badges.push('Safety Star');
    if (driver.trips_completed >= 50) badges.push('Road Warrior');
    if (driver.improvement_since_last_period >= 10) badges.push('Most Improved');
    
    return badges;
  };

  // Get rank icon
  const getRankIcon = (rank: number) => {
    if (rank === 1) return <EmojiEvents sx={{ color: '#ffd700' }} />;
    if (rank === 2) return <EmojiEvents sx={{ color: '#c0c0c0' }} />;
    if (rank === 3) return <EmojiEvents sx={{ color: '#cd7f32' }} />;
    return <Typography variant="body2" sx={{ fontWeight: 'bold' }}>#{rank}</Typography>;
  };

  // Format numbers
  const formatNumber = (num: number, decimals = 1) => 
    num.toLocaleString(undefined, { maximumFractionDigits: decimals });

  if (loading) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>Driver Efficiency Leaderboard</Typography>
        <LinearProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>Driver Efficiency Leaderboard</Typography>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  if (!leaderboard?.drivers.length) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>Driver Efficiency Leaderboard</Typography>
        <Alert severity="info">No driver data available for the selected period.</Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ p: { xs: 2, sm: 3 } }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" sx={{ fontSize: { xs: '1.5rem', sm: '2rem', md: '2.125rem' } }}>
          Driver Efficiency Leaderboard
        </Typography>
        <FormControl size="small" sx={{ minWidth: 120 }}>
          <InputLabel>Period</InputLabel>
          <Select
            value={selectedPeriod}
            onChange={(e) => setSelectedPeriod(e.target.value as any)}
            label="Period"
          >
            <MenuItem value="daily">Daily</MenuItem>
            <MenuItem value="weekly">Weekly</MenuItem>
            <MenuItem value="monthly">Monthly</MenuItem>
            <MenuItem value="yearly">Yearly</MenuItem>
          </Select>
        </FormControl>
      </Box>

      {/* Summary Cards */}
      {summary && (
        <Grid container spacing={{ xs: 2, sm: 3 }} sx={{ mb: 4 }}>
          <Grid item xs={6} sm={3}>
            <Card>
              <CardContent sx={{ p: { xs: 1.5, sm: 2 } }}>
                <Typography variant="h6" color="primary" sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
                  {summary.totalDrivers}
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>
                  Active Drivers
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={6} sm={3}>
            <Card>
              <CardContent sx={{ p: { xs: 1.5, sm: 2 } }}>
                <Typography variant="h6" color="success.main" sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
                  {summary.avgEfficiency}%
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>
                  Avg Efficiency
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={6} sm={3}>
            <Card>
              <CardContent sx={{ p: { xs: 1.5, sm: 2 } }}>
                <Typography variant="h6" color="info.main" sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
                  {summary.avgFuelEfficiency}
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>
                  Avg Fuel Efficiency (km/L)
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={6} sm={3}>
            <Card>
              <CardContent sx={{ p: { xs: 1.5, sm: 2 } }}>
                <Typography variant="h6" color="warning.main" sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
                  {formatNumber(summary.totalDistance / 1000)}k
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>
                  Total Distance (km)
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {/* Charts */}
      <Grid container spacing={{ xs: 2, sm: 3 }} sx={{ mb: 4 }}>
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: { xs: 1.5, sm: 2 } }}>
            <Typography variant="h6" gutterBottom sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
              Efficiency Distribution
            </Typography>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={efficiencyChartData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ range, count }) => `${range}: ${count}`}
                  outerRadius={80}
                  fill="#8884d8"
                  dataKey="count"
                >
                  {efficiencyChartData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <RechartsTooltip />
              </PieChart>
            </ResponsiveContainer>
          </Paper>
        </Grid>
        
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: { xs: 1.5, sm: 2 } }}>
            <Typography variant="h6" gutterBottom sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
              Top 10 Drivers by Efficiency
            </Typography>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={leaderboard.drivers.slice(0, 10)}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis 
                  dataKey="driver_name" 
                  angle={-45}
                  textAnchor="end"
                  height={80}
                  fontSize={10}
                  interval={0}
                />
                <YAxis fontSize={12} />
                <RechartsTooltip 
                  formatter={(value, name) => [value, 'Efficiency Score']}
                  labelFormatter={(label) => `Driver: ${label}`}
                />
                <Bar dataKey="efficiency_score" fill="#1976d2" />
              </BarChart>
            </ResponsiveContainer>
          </Paper>
        </Grid>
      </Grid>

      {/* Leaderboard Table */}
      <Paper sx={{ p: { xs: 1.5, sm: 2 } }}>
        <Typography variant="h6" gutterBottom sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
          Driver Rankings
        </Typography>
        <TableContainer sx={{ overflowX: 'auto' }}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>Rank</TableCell>
                <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>Driver</TableCell>
                <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>Efficiency Score</TableCell>
                <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>Fuel Efficiency</TableCell>
                <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>Distance</TableCell>
                <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>Trips</TableCell>
                <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>Improvement</TableCell>
                <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>Badges</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {leaderboard.drivers.map((driver, index) => {
                const badges = getBadges(driver);
                return (
                  <TableRow key={driver.driver_id} hover>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        {getRankIcon(driver.rank)}
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Avatar sx={{ width: 32, height: 32, fontSize: '0.875rem' }}>
                          {driver.driver_name.charAt(0)}
                        </Avatar>
                        <Box>
                          <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
                            {driver.driver_name}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            ID: {driver.driver_id.slice(0, 8)}...
                          </Typography>
                        </Box>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <LinearProgress
                          variant="determinate"
                          value={driver.efficiency_score}
                          sx={{ width: 60, height: 8, borderRadius: 4 }}
                        />
                        <Typography variant="body2" sx={{ minWidth: 35 }}>
                          {driver.efficiency_score}%
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                        <LocalGasStation fontSize="small" color="primary" />
                        <Typography variant="body2">
                          {formatNumber(driver.fuel_efficiency)} km/L
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {formatNumber(driver.total_distance)} km
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {driver.trips_completed}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                        {driver.improvement_since_last_period > 0 ? (
                          <TrendingUp color="success" fontSize="small" />
                        ) : driver.improvement_since_last_period < 0 ? (
                          <TrendingDown color="error" fontSize="small" />
                        ) : null}
                        <Typography 
                          variant="body2" 
                          color={driver.improvement_since_last_period > 0 ? 'success.main' : 
                                 driver.improvement_since_last_period < 0 ? 'error.main' : 'text.secondary'}
                        >
                          {driver.improvement_since_last_period > 0 ? '+' : ''}
                          {formatNumber(driver.improvement_since_last_period)}%
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                        {badges.slice(0, 2).map((badge, badgeIndex) => (
                          <Chip
                            key={badgeIndex}
                            label={badge}
                            size="small"
                            color="primary"
                            variant="outlined"
                            sx={{ fontSize: '0.7rem', height: 20 }}
                          />
                        ))}
                        {badges.length > 2 && (
                          <Tooltip title={badges.slice(2).join(', ')}>
                            <Chip
                              label={`+${badges.length - 2}`}
                              size="small"
                              color="default"
                              variant="outlined"
                              sx={{ fontSize: '0.7rem', height: 20 }}
                            />
                          </Tooltip>
                        )}
                      </Box>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>
    </Box>
  );
};

export default DriverEfficiencyLeaderboard;
