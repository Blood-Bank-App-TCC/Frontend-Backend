import { Building2, Edit, Plus, Save, Trash2, X } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import { formatIndonesianPhone } from "../../models/phone";
import type { Hospital } from "../../models/types";
import StatusBadge from "../components/StatusBadge";
import PageHeader from "../layout/PageHeader";

const emptyForm = {
  name: "",
  address: "",
  picName: "",
  picPhone: "",
  email: "",
  latitude: -7.7839,
  longitude: 110.3798,
};

export default function HospitalsPage() {
  const [hospitals, setHospitals] = useState<Hospital[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState(emptyForm);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  async function load() {
    setLoading(true);
    try {
      setHospitals(await bankDarahController.hospitals());
    } catch (err) {
      setError(apiErrorMessage(err, "Daftar rumah sakit gagal dimuat."));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  function openCreate() {
    setForm(emptyForm);
    setEditingId(null);
    setShowForm(true);
    setError("");
    setMessage("");
  }

  function openEdit(hospital: Hospital) {
    setForm({
      name: hospital.name,
      address: hospital.address,
      picName: hospital.picName,
      picPhone: formatIndonesianPhone(hospital.picPhone),
      email: hospital.email ?? "",
      latitude: hospital.latitude,
      longitude: hospital.longitude,
    });
    setEditingId(hospital.id);
    setShowForm(true);
    setError("");
    setMessage("");
  }

  function closeForm() {
    setShowForm(false);
    setEditingId(null);
    setForm(emptyForm);
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError("");
    setMessage("");
    try {
      if (editingId) {
        await bankDarahController.updateHospital(editingId, form);
        setMessage("Data rumah sakit berhasil diperbarui.");
      } else {
        await bankDarahController.createHospital(form);
        setMessage("Rumah sakit berhasil ditambahkan.");
      }
      closeForm();
      await load();
    } catch (err) {
      setError(apiErrorMessage(err, editingId ? "Data rumah sakit gagal diperbarui." : "Rumah sakit gagal ditambahkan."));
    } finally {
      setSaving(false);
    }
  }

  async function deleteHospital(hospital: Hospital) {
    const confirmed = window.confirm(`Hapus ${hospital.name} dari daftar rumah sakit mitra?`);
    if (!confirmed) return;

    setDeletingId(hospital.id);
    setError("");
    setMessage("");
    try {
      await bankDarahController.deleteHospital(hospital.id);
      if (editingId === hospital.id) {
        closeForm();
      }
      setMessage("Rumah sakit berhasil dihapus.");
      await load();
    } catch (err) {
      setError(apiErrorMessage(err, "Rumah sakit gagal dihapus."));
    } finally {
      setDeletingId(null);
    }
  }

  return (
    <>
      <PageHeader
        eyebrow="Master data"
        title="Rumah Sakit Mitra"
        action={
          <button className="btn primary" type="button" onClick={openCreate}>
            <Plus size={18} />
            Tambah RS
          </button>
        }
      />
      {error && <p className="alert danger">{error}</p>}
      {message && <p className="alert success">{message}</p>}
      {showForm && (
        <form className="panel form-grid" onSubmit={submit}>
          <div className="panel-header span-2">
            <h2>{editingId ? "Edit Rumah Sakit" : "Tambah Rumah Sakit"}</h2>
            <button className="icon-button" title="Tutup form" type="button" onClick={closeForm}>
              <X size={18} />
            </button>
          </div>
          <label>
            Nama Rumah Sakit
            <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
          </label>
          <label>
            PIC
            <input value={form.picName} onChange={(event) => setForm({ ...form, picName: event.target.value })} required />
          </label>
          <label>
            Telepon PIC
            <input
              inputMode="tel"
              maxLength={15}
              pattern="^\+62[0-9]{1,12}$"
              title="Nomor harus diawali +62 dan maksimal 12 digit setelah kode negara."
              value={form.picPhone}
              onChange={(event) => setForm({ ...form, picPhone: formatIndonesianPhone(event.target.value) })}
              required
            />
          </label>
          <label>
            Email
            <input type="email" value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} />
          </label>
          <label className="span-2">
            Alamat
            <input value={form.address} onChange={(event) => setForm({ ...form, address: event.target.value })} />
          </label>
          <label>
            Latitude
            <input type="number" step="0.0001" value={form.latitude} onChange={(event) => setForm({ ...form, latitude: Number(event.target.value) })} />
          </label>
          <label>
            Longitude
            <input type="number" step="0.0001" value={form.longitude} onChange={(event) => setForm({ ...form, longitude: Number(event.target.value) })} />
          </label>
          <div className="form-actions span-2">
            <button className="btn primary" type="submit" disabled={saving}>
              <Save size={18} />
              {saving ? "Menyimpan..." : editingId ? "Simpan Perubahan" : "Simpan Rumah Sakit"}
            </button>
          </div>
        </form>
      )}
      {loading ? (
        <p className="subtle">Memuat rumah sakit...</p>
      ) : (
        <section className="hospital-grid">
          {hospitals.map((hospital) => (
            <article className="hospital-card" key={hospital.id}>
              <div className="hospital-card__top">
                <Building2 size={24} />
                <div className="hospital-card__actions">
                  <button className="icon-button" title="Edit rumah sakit" type="button" onClick={() => openEdit(hospital)}>
                    <Edit size={18} />
                  </button>
                  <button
                    className="icon-button danger"
                    disabled={deletingId === hospital.id}
                    title="Hapus rumah sakit"
                    type="button"
                    onClick={() => deleteHospital(hospital)}
                  >
                    <Trash2 size={18} />
                  </button>
                </div>
              </div>
              <h2>{hospital.name}</h2>
              <p>{hospital.address || "-"}</p>
              <span>{hospital.picName || "-"}</span>
              <span>{hospital.picPhone || "-"}</span>
              <StatusBadge variant={hospital.isActive ? "safe" : "critical"}>{hospital.isActive ? "Aktif" : "Nonaktif"}</StatusBadge>
            </article>
          ))}
          {hospitals.length === 0 && <p className="empty">Belum ada rumah sakit mitra.</p>}
        </section>
      )}
    </>
  );
}
