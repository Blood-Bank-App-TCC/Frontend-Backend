import { Link } from "react-router-dom";
import type { BloodStock, Donor, EmergencyRequest } from "../../models/types";
import { formatDateTime, stockLevel } from "../../models/status";
import StatusBadge from "./StatusBadge";

export function RequestTable({ requests }: { requests: EmergencyRequest[] }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>RS</th>
            <th>Darah</th>
            <th>Jumlah</th>
            <th>Urgensi</th>
            <th>Eligible</th>
            <th>Status</th>
            <th>Waktu</th>
          </tr>
        </thead>
        <tbody>
          {requests.map((request) => (
            <tr key={request.id}>
              <td>
                <Link to={`/emergency/${request.id}/${request.broadcastId ? "monitor" : "broadcast"}`}>{request.hospitalName}</Link>
              </td>
              <td>
                {request.bloodType} / {request.productType}
              </td>
              <td>{request.quantityNeeded}</td>
              <td>
                <StatusBadge urgency={request.urgencyLevel} />
              </td>
              <td>{request.eligibleCount}</td>
              <td>{request.status}</td>
              <td>{formatDateTime(request.createdAt)}</td>
            </tr>
          ))}
          {requests.length === 0 && (
            <tr>
              <td colSpan={7}>Belum ada permintaan.</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

export function DonorTable({ donors }: { donors: Donor[] }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Nama</th>
            <th>Golongan</th>
            <th>Jarak</th>
            <th>Kontak</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {donors.map((donor) => (
            <tr key={donor.id}>
              <td>
                <Link to={`/donors/${donor.id}`}>{donor.fullName}</Link>
              </td>
              <td>{donor.bloodType}</td>
              <td>{donor.distanceKm.toFixed(1)} km</td>
              <td>{donor.phone}</td>
              <td>
                <StatusBadge variant={donor.isEligible ? "safe" : "low"}>{donor.isEligible ? "Eligible" : "Jeda"}</StatusBadge>
              </td>
            </tr>
          ))}
          {donors.length === 0 && (
            <tr>
              <td colSpan={5}>Tidak ada donor eligible.</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

export function StockTable({ stock }: { stock: BloodStock[] }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Golongan</th>
            <th>Produk</th>
            <th>Jumlah</th>
            <th>Threshold</th>
            <th>Status</th>
            <th>Update</th>
            <th>Keterangan</th>
          </tr>
        </thead>
        <tbody>
          {stock.map((item) => (
            <tr key={item.id}>
              <td>{item.bloodType}</td>
              <td>{item.productType}</td>
              <td>{item.quantity}</td>
              <td>
                {item.criticalThreshold} / {item.safeThreshold}
              </td>
              <td>
                <StatusBadge variant={stockLevel(item) === "critical" ? "critical" : stockLevel(item) === "low" ? "low" : "safe"}>
                  {stockLevel(item) === "critical" ? "Kritis" : stockLevel(item) === "low" ? "Menipis" : "Aman"}
                </StatusBadge>
              </td>
              <td>{formatDateTime(item.updatedAt)}</td>
              <td>{item.description || "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
