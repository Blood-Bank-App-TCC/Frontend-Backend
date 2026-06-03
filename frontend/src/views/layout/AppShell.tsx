import { Activity, BarChart3, Building2, Home, LogOut, PackagePlus, QrCode, ShieldCheck, Siren, Users } from "lucide-react";
import { useEffect } from "react";
import { Link, NavLink, Navigate, Route, Routes } from "react-router-dom";
import { bankDarahController } from "../../controllers/bankDarahController";
import { useAuthStore } from "../../models/authStore";
import BroadcastPage from "../pages/BroadcastPage";
import CheckinPage from "../pages/CheckinPage";
import DashboardPage from "../pages/DashboardPage";
import DonorDetailPage from "../pages/DonorDetailPage";
import DonorsPage from "../pages/DonorsPage";
import EmergencyNewPage from "../pages/EmergencyNewPage";
import HospitalsPage from "../pages/HospitalsPage";
import MonitorPage from "../pages/MonitorPage";
import ReportsPage from "../pages/ReportsPage";
import StockPage from "../pages/StockPage";

const navItems = [
  { to: "/dashboard", label: "Dashboard", icon: Home },
  { to: "/emergency/new", label: "Permintaan", icon: Siren },
  { to: "/checkin", label: "Check-in QR", icon: QrCode },
  { to: "/stock", label: "Stok Darah", icon: PackagePlus },
  { to: "/donors", label: "Pendonor", icon: Users },
  { to: "/hospitals", label: "Rumah Sakit", icon: Building2 },
  { to: "/reports", label: "Laporan", icon: BarChart3 },
];

export default function AppShell() {
  const token = useAuthStore((state) => state.token);
  const user = useAuthStore((state) => state.user);
  const setSession = useAuthStore((state) => state.setSession);
  const clearSession = useAuthStore((state) => state.clearSession);

  useEffect(() => {
    if (!token) return;
    bankDarahController.me().then((admin) => setSession(token, admin)).catch(clearSession);
  }, [clearSession, setSession, token]);

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <Link className="brand" to="/dashboard">
          <span className="brand-mark">
            <Activity size={24} />
          </span>
          <span>
            <strong>Bank Darah</strong>
            <small>Admin PMI</small>
          </span>
        </Link>
        <nav className="nav-list">
          {navItems.map((item) => (
            <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} key={item.to} to={item.to}>
              <item.icon size={18} />
              <span>{item.label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-user">
          <ShieldCheck size={18} />
          <div>
            <strong>{user?.fullName}</strong>
            <small>{user?.role}</small>
          </div>
          <button className="icon-button" title="Keluar" onClick={clearSession} type="button">
            <LogOut size={18} />
          </button>
        </div>
      </aside>
      <main className="main-content">
        <Routes>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/emergency/new" element={<EmergencyNewPage />} />
          <Route path="/emergency/:id/broadcast" element={<BroadcastPage />} />
          <Route path="/emergency/:id/monitor" element={<MonitorPage />} />
          <Route path="/checkin" element={<CheckinPage />} />
          <Route path="/stock" element={<StockPage />} />
          <Route path="/donors" element={<DonorsPage />} />
          <Route path="/donors/:id" element={<DonorDetailPage />} />
          <Route path="/hospitals" element={<HospitalsPage />} />
          <Route path="/reports" element={<ReportsPage />} />
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </main>
    </div>
  );
}
