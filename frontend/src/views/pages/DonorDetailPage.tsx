import { ArrowLeft, Edit, Power, Save, X } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { Donor } from "../../models/types";
import { formatDate } from "../../models/status";
import Loading from "../components/Loading";
import StatusBadge from "../components/StatusBadge";
import PageHeader from "../layout/PageHeader";

export default function DonorDetailPage() {
  const { id = "" } = useParams();
  const [donor, setDonor] = useState<Donor | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showEdit, setShowEdit] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [editForm, setEditForm] = useState({
    fullName: "",
    phone: "",
    email: "",
    address: "",
  });

  useEffect(() => {
    setLoading(true);
    setError("");
    bankDarahController
      .donor(id)
      .then((d) => {
        setDonor(d);
        setEditForm({
          fullName: d.fullName,
          phone: d.phone,
          email: d.email ?? "",
          address: d.address,
        });
      })
      .catch((err) => setError(apiErrorMessage(err, "Profil pendonor gagal dimuat.")))
      .finally(() => setLoading(false));
  }, [id]);

  async function toggleStatus() {
    if (!donor) return;
    setError("");
    setMessage("");
    setSaving(true);
    try {
      const updated = await bankDarahController.updateDonorStatus(donor.id, !donor.isActive);
      setDonor(updated);
      setMessage(updated.isActive ? "Pendonor berhasil diaktifkan." : "Pendonor berhasil dinonaktifkan.");
    } catch (err) {
      setError(apiErrorMessage(err, "Status pendonor gagal diperbarui."));
    } finally {
      setSaving(false);
    }
  }

  async function submitEdit(event: FormEvent) {
    event.preventDefault();
    if (!donor) return;
    setError("");
    setMessage("");
    setSaving(true);
    try {
      const updated = await bankDarahController.updateDonor(donor.id, editForm);
      setDonor(updated);
      setShowEdit(false);
      setMessage("Data pendonor berhasil diperbarui.");
    } catch (err) {
      setError(apiErrorMessage(err, "Data pendonor gagal diperbarui."));
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <Loading title="Memuat profil donor" />;
  if (!donor) return <p className="alert danger">{error || "Pendonor tidak ditemukan."}</p>;

  return (
    <>
      <PageHeader
        eyebrow="Profil pendonor"
        title={donor.fullName}
        action={
          <div className="header-actions">
            <Link className="btn secondary" to="/donors">
              <ArrowLeft size={18} />
              Kembali
            </Link>
            <button className="btn secondary" type="button" onClick={() => setShowEdit((v) => !v)}>
              {showEdit ? <X size={18} /> : <Edit size={18} />}
              {showEdit ? "Batal Edit" : "Edit Data"}
            </button>
            <button className="btn danger" disabled={saving} onClick={toggleStatus} type="button">
              <Power size={18} />
              {donor.isActive ? "Nonaktifkan" : "Aktifkan"}
            </button>
          </div>
        }
      />

      {error && <p className="alert danger">{error}</p>}
      {message && <p className="alert success">{message}</p>}

      {showEdit && (
        <form className="panel form-grid" onSubmit={submitEdit}>
          <label>
            Nama Lengkap
            <input
              value={editForm.fullName}
              onChange={(e) => setEditForm({ ...editForm, fullName: e.target.value })}
              required
            />
          </label>
          <label>
            Nomor Telepon
            <input
              value={editForm.phone}
              onChange={(e) => setEditForm({ ...editForm, phone: e.target.value })}
              required
            />
          </label>
          <label>
            Email
            <input
              type="email"
              value={editForm.email}
              onChange={(e) => setEditForm({ ...editForm, email: e.target.value })}
            />
          </label>
          <label className="span-2">
            Alamat
            <input
              value={editForm.address}
              onChange={(e) => setEditForm({ ...editForm, address: e.target.value })}
            />
          </label>
          <div className="form-actions span-2">
            <button className="btn primary" type="submit" disabled={saving}>
              <Save size={16} />
              {saving ? "Menyimpan..." : "Simpan Perubahan"}
            </button>
          </div>
        </form>
      )}

      <section className="panel profile-detail">
        <div className="avatar large">{donor.bloodType}</div>
        <div>
          <dl>
            <div>
              <dt>Status Donor</dt>
              <dd>
                <StatusBadge variant={donor.isEligible ? "safe" : "low"}>{donor.isEligible ? "Eligible" : "Jeda donor"}</StatusBadge>
              </dd>
            </div>
            <div>
              <dt>Status Akun</dt>
              <dd>
                <StatusBadge variant={donor.isActive ? "safe" : "critical"}>{donor.isActive ? "Aktif" : "Nonaktif"}</StatusBadge>
              </dd>
            </div>
            <div>
              <dt>Telepon</dt>
              <dd>{donor.phone}</dd>
            </div>
            <div>
              <dt>Email</dt>
              <dd>{donor.email ?? "-"}</dd>
            </div>
            <div>
              <dt>Alamat</dt>
              <dd>{donor.address}</dd>
            </div>
            <div>
              <dt>Token QR</dt>
              <dd>{donor.uuid}</dd>
            </div>
            <div>
              <dt>Donor Terakhir</dt>
              <dd>{formatDate(donor.lastDonation)}</dd>
            </div>
            <div>
              <dt>Eligible Berikutnya</dt>
              <dd>{formatDate(donor.nextEligible)}</dd>
            </div>
          </dl>
        </div>
      </section>
      <section className="panel">
        <div className="panel-header">
          <h2>Riwayat Donasi</h2>
          <span className="subtle">{donor.donationHistory?.length ?? 0} catatan</span>
        </div>
        <div className="history-list">
          {donor.donationHistory?.map((record) => (
            <div key={record.id}>
              <strong>{formatDate(record.date)}</strong>
              <span>
                {record.location} - {record.status}
              </span>
              <small>
                TD {record.bloodPressure}, Hb {record.hemoglobin}, BB {record.weight} kg
              </small>
            </div>
          ))}
        </div>
      </section>
    </>
  );
}
