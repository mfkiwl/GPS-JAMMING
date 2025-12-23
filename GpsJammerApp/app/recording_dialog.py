from PySide6.QtWidgets import (QDialog, QVBoxLayout, QHBoxLayout, 
                             QLabel, QDoubleSpinBox, QSpinBox, 
                             QPushButton, QGroupBox, QGridLayout, QTextEdit, QCheckBox, QLineEdit,
                             QScrollArea, QWidget)
from PySide6.QtCore import Qt, QTimer, Signal, QObject
import subprocess
import threading
import time
import re


class RecordingDialog(QDialog):
    log_signal = Signal(str)
    warmup_complete = Signal()
    update_timer = Signal(int)
    devices_detected = Signal(list)
    
    def __init__(self, parent=None):
        super().__init__(parent)
        self.setWindowTitle("Nagrywanie danych SDR - GPS Jammer Detection")
        self.setModal(True)
        self.resize(700, 750)
        
        self.recording_process = None
        self.recording_processes = []
        self.is_recording = False
        self.warmup_timer = None
        self.recording_start_time = None
        
        self.detected_devices = []
        self.device_checkboxes = []
        self.device_biast_checkboxes = []
        
        self.log_signal.connect(self._log_message_safe)
        self.warmup_complete.connect(self._warmup_finished)
        self.update_timer.connect(self._update_timer_display)
        self.devices_detected.connect(self._update_devices_ui)
        
        self.setStyleSheet("""
        QDialog {
            background-color: #ecf0f1;
        }
        QLabel {
            color: #2c3e50;
            font-size: 12px;
            font-weight: bold;
        }
        QGroupBox {
            font-weight: bold;
            font-size: 14px;
            border: 2px solid #bdc3c7;
            border-radius: 10px;
            margin-top: 10px;
            padding-top: 10px;
            background-color: #f8f9fa;
        }
        QGroupBox::title {
            subcontrol-origin: margin;
            left: 10px;
            padding: 0 8px 0 8px;
            color: #2c3e50;
            background-color: #f8f9fa;
        }
        """)
        
        layout = QVBoxLayout(self)
        layout.setContentsMargins(15, 15, 15, 15)
        layout.setSpacing(10)

        params_group = QGroupBox("Parametry nagrywania")
        params_layout = QGridLayout(params_group)
        params_layout.setVerticalSpacing(10)
        params_layout.setHorizontalSpacing(10)
        params_layout.addWidget(QLabel("Częstotliwość:"), 0, 0)
        self.frequency_spin = QDoubleSpinBox()
        self.frequency_spin.setRange(1000.0, 2000.0)
        self.frequency_spin.setValue(1575.42)
        self.frequency_spin.setDecimals(2)
        self.frequency_spin.setSuffix(" MHz")
        self.frequency_spin.setStyleSheet(self.get_spinbox_style())
        params_layout.addWidget(self.frequency_spin, 0, 1)
        
        params_layout.addWidget(QLabel("Częstotliwość próbkowania:"), 1, 0)
        self.sample_rate_spin = QDoubleSpinBox()
        self.sample_rate_spin.setRange(0.1, 20.0)
        self.sample_rate_spin.setDecimals(3)
        self.sample_rate_spin.setSingleStep(0.001)
        self.sample_rate_spin.setValue(round(2048.0 / 1000.0, 3))
        self.sample_rate_spin.setSuffix(" MHz")
        self.sample_rate_spin.setStyleSheet(self.get_spinbox_style())
        params_layout.addWidget(self.sample_rate_spin, 1, 1)
        
        params_layout.addWidget(QLabel("Nazwa pliku:"), 2, 0)
        self.filename_edit = QLineEdit()
        self.filename_edit.setText("test.bin")
        self.filename_edit.setStyleSheet("""
        QLineEdit {
            border: 2px solid #bdc3c7;
            border-radius: 5px;
            padding: 8px;
            font-size: 13px;
            background-color: white;
            color: #2c3e50;
            min-width: 120px;
        }
        QLineEdit:focus {
            border-color: #3498db;
        }
        QLineEdit:disabled {
            background-color: #ecf0f1;
            color: #95a5a6;
            border-color: #bdc3c7;
        }
        """)
        params_layout.addWidget(self.filename_edit, 2, 1)
        
        layout.addWidget(params_group)
        
        devices_group = QGroupBox("Wykryte urządzenia SDR")
        devices_layout = QVBoxLayout(devices_group)
        devices_layout.setSpacing(8)
        
        self.scan_btn = QPushButton("Skanuj urządzenia")
        self.scan_btn.clicked.connect(self.scan_devices)
        self.scan_btn.setStyleSheet("""
        QPushButton {
            background-color: #3498db;
            color: white;
            border: none;
            border-radius: 5px;
            padding: 8px 12px;
            font-size: 12px;
            font-weight: bold;
        }
        QPushButton:hover {
            background-color: #2980b9;
        }
        QPushButton:pressed {
            background-color: #1f618d;
        }
        QPushButton:disabled {
            background-color: #95a5a6;
            color: #bdc3c7;
        }
        """)
        devices_layout.addWidget(self.scan_btn)
        
        self.devices_scroll = QScrollArea()
        self.devices_scroll.setWidgetResizable(True)
        self.devices_scroll.setMinimumHeight(140)
        self.devices_scroll.setMaximumHeight(165)
        self.devices_scroll.setStyleSheet("""
        QScrollArea {
            border: 2px solid #bdc3c7;
            border-radius: 8px;
            background-color: white;
        }
        """)
        
        self.devices_widget = QWidget()
        self.devices_widget_layout = QVBoxLayout(self.devices_widget)
        self.devices_widget_layout.setAlignment(Qt.AlignTop)
        
        self.no_devices_label = QLabel("Nie wykryto urzadzen RTL-SDR")
        self.no_devices_label.setStyleSheet("""
        QLabel {
            color: #e74c3c;
            font-size: 12px;
            font-weight: bold;
            padding: 15px;
        }
        """)
        self.no_devices_label.setAlignment(Qt.AlignCenter)
        self.devices_widget_layout.addWidget(self.no_devices_label)
        
        self.devices_scroll.setWidget(self.devices_widget)
        devices_layout.addWidget(self.devices_scroll)
        
        layout.addWidget(devices_group)
        
        control_layout = QHBoxLayout()
        control_layout.setSpacing(10)

        self.warmup_btn = QPushButton("Nagrzej odbiornik (60s)")
        self.warmup_btn.clicked.connect(self.warmup_receiver)
        self.warmup_btn.setEnabled(False)
        self.warmup_btn.setStyleSheet("""
        QPushButton {
            background-color: #f39c12;
            color: white;
            border: none;
            border-radius: 5px;
            padding: 10px 15px;
            font-size: 12px;
            font-weight: bold;
        }
        QPushButton:hover {
            background-color: #e67e22;
        }
        QPushButton:pressed {
            background-color: #d68910;
        }
        QPushButton:disabled {
            background-color: #95a5a6;
            color: #bdc3c7;
        }
        """)
        control_layout.addWidget(self.warmup_btn)
        
        self.record_toggle_btn = QPushButton("Wlacz nagrywanie")
        self.record_toggle_btn.clicked.connect(self.toggle_recording)
        self.record_toggle_btn.setEnabled(False)
        self.record_toggle_btn.setStyleSheet("""
        QPushButton {
            background-color: #27ae60;
            color: white;
            border: none;
            border-radius: 5px;
            padding: 10px 15px;
            font-size: 12px;
            font-weight: bold;
        }
        QPushButton:hover {
            background-color: #2ecc71;
        }
        QPushButton:pressed {
            background-color: #229954;
        }
        QPushButton:disabled {
            background-color: #95a5a6;
            color: #bdc3c7;
        }
        """)
        control_layout.addWidget(self.record_toggle_btn)
        
        layout.addLayout(control_layout)
        
        self.timer_label = QLabel("Czas nagrywania: 00:00")
        self.timer_label.setStyleSheet("""
        QLabel {
            color: #2c3e50;
            font-size: 14px;
            font-weight: bold;
            font-family: 'Courier New', monospace;
            padding: 8px;
            background-color: #ecf0f1;
            border: 2px solid #bdc3c7;
            border-radius: 5px;
        }
        """)
        self.timer_label.setAlignment(Qt.AlignCenter)
        layout.addWidget(self.timer_label)
        
        logs_group = QGroupBox("Logi")
        logs_layout = QVBoxLayout(logs_group)
        logs_layout.setContentsMargins(5, 5, 5, 5)
        
        self.logs_text = QTextEdit()
        self.logs_text.setReadOnly(True)
        self.logs_text.setMinimumHeight(150)
        self.logs_text.setMaximumHeight(200)
        self.logs_text.setPlainText("Gotowy do nagrywania...\n")
        self.logs_text.setStyleSheet("""
        QTextEdit {
            border: 2px solid #bdc3c7;
            border-radius: 5px;
            padding: 8px;
            font-family: 'Courier New', monospace;
            font-size: 10px;
            background-color: #2c3e50;
            color: #ecf0f1;
        }
        """)
        logs_layout.addWidget(self.logs_text)
        
        layout.addWidget(logs_group)
        
        self.close_btn = QPushButton("Zamknij")
        self.close_btn.clicked.connect(self.close_dialog)
        self.close_btn.setStyleSheet("""
        QPushButton {
            background-color: #e74c3c;
            color: white;
            border: none;
            border-radius: 5px;
            padding: 10px 15px;
            font-size: 12px;
            font-weight: bold;
        }
        QPushButton:hover {
            background-color: #c0392b;
        }
        QPushButton:pressed {
            background-color: #a93226;
        }
        """)
        layout.addWidget(self.close_btn)
        
        QTimer.singleShot(500, self.scan_devices)
    
    def scan_devices(self):
        self.scan_btn.setEnabled(False)
        self.scan_btn.setText("Skanowanie...")
        self.log_message("Rozpoczynam skanowanie urzadzen RTL-SDR...")
        
        def scan_thread():
            devices = []
            try:
                result = subprocess.run(['rtl_test', '-t'], 
                                      capture_output=True, 
                                      text=True,
                                      timeout=5)
                
                output = result.stdout + result.stderr
                self.log_signal.emit(f"Output rtl_test:\n{output}")
                
                lines = output.split('\n')
                for i, line in enumerate(lines):
                    match = re.match(r'\s*(\d+):\s*(.+)', line)
                    if match:
                        device_idx = int(match.group(1))
                        device_info = match.group(2).strip()
                        
                        serial_match = re.search(r'SN:\s*(\S+)', device_info)
                        serial = serial_match.group(1) if serial_match else f"Device_{device_idx}"
                        
                        name = re.sub(r',?\s*SN:\s*\S+', '', device_info).strip()
                        
                        devices.append({
                            'index': device_idx,
                            'name': name,
                            'serial': serial
                        })
                        self.log_signal.emit(f"Znaleziono: [{device_idx}] {name} (SN: {serial})")
                
                if not devices:
                    self.log_signal.emit("Nie wykryto urzadzen RTL-SDR")
                    self.log_signal.emit("Upewnij sie, ze urzadzenie jest podlaczone")
                else:
                    self.log_signal.emit(f"Wykryto {len(devices)} urzadzenie(n)")
                
            except FileNotFoundError:
                self.log_signal.emit("BLAD: Nie znaleziono komendy rtl_test")
                self.log_signal.emit("Zainstaluj: sudo apt install rtl-sdr")
            except subprocess.TimeoutExpired:
                self.log_signal.emit("BLAD: Timeout podczas skanowania")
            except Exception as e:
                self.log_signal.emit(f"BLAD: {str(e)}")
            
            self.devices_detected.emit(devices)
        
        thread = threading.Thread(target=scan_thread)
        thread.daemon = True
        thread.start()
    
    def _update_devices_ui(self, devices):
        self.detected_devices = devices
        self.scan_btn.setEnabled(True)
        self.scan_btn.setText("Skanuj urządzenia")
        
        while self.devices_widget_layout.count():
            child = self.devices_widget_layout.takeAt(0)
            if child.widget():
                child.widget().deleteLater()
        
        self.device_checkboxes = []
        self.device_biast_checkboxes = []
        
        if not devices:
            no_devices = QLabel("Nie wykryto urzadzen RTL-SDR")
            no_devices.setStyleSheet("""
            QLabel {
                color: #e74c3c;
                font-size: 12px;
                font-weight: bold;
                padding: 15px;
            }
            """)
            no_devices.setAlignment(Qt.AlignCenter)
            self.devices_widget_layout.addWidget(no_devices)
            
            self.warmup_btn.setEnabled(False)
            self.record_toggle_btn.setEnabled(False)
        else:
            for device in devices:
                device_frame = QWidget()
                device_frame.setStyleSheet("""
                QWidget {
                    background-color: #ecf0f1;
                    border-radius: 5px;
                    padding: 3px;
                    margin: 2px;
                }
                """)
                device_layout = QHBoxLayout(device_frame)
                device_layout.setContentsMargins(8, 3, 8, 3)
                
                device_cb = QCheckBox()
                device_cb.setChecked(True)
                device_cb.stateChanged.connect(self._update_button_states)
                device_cb.setStyleSheet("""
                QCheckBox::indicator {
                    width: 18px;
                    height: 18px;
                    border: 2px solid #bdc3c7;
                    border-radius: 3px;
                    background-color: white;
                }
                QCheckBox::indicator:checked {
                    background-color: #27ae60;
                    border-color: #27ae60;
                }
                QCheckBox::indicator:unchecked {
                    background-color: white;
                    border-color: #bdc3c7;
                }
                """)
                self.device_checkboxes.append((device['index'], device_cb))
                device_layout.addWidget(device_cb)
                
                device_label = QLabel(f"[{device['index']}] {device['name']} (SN: {device['serial']})")
                device_label.setStyleSheet("""
                QLabel {
                    color: #2c3e50;
                    font-size: 11px;
                    font-weight: normal;
                }
                """)
                device_layout.addWidget(device_label, stretch=1)
                
                biast_cb = QCheckBox("BiasT")
                biast_cb.setChecked(False)
                biast_cb.stateChanged.connect(lambda state, idx=device['index'], cb=biast_cb: 
                                             self.toggle_biast_for_device(idx, cb.isChecked()))
                biast_cb.setStyleSheet("""
                QCheckBox {
                    font-size: 10px;
                    color: #34495e;
                    font-weight: bold;
                }
                QCheckBox::indicator {
                    width: 14px;
                    height: 14px;
                    border: 2px solid #bdc3c7;
                    border-radius: 3px;
                    background-color: white;
                }
                QCheckBox::indicator:checked {
                    background-color: #f39c12;
                    border-color: #f39c12;
                }
                """)
                self.device_biast_checkboxes.append((device['index'], biast_cb))
                device_layout.addWidget(biast_cb)
                
                self.devices_widget_layout.addWidget(device_frame)
            
            self._update_button_states()
    
    def _update_button_states(self):
        selected = self.get_selected_devices()
        has_selection = len(selected) > 0
        
        self.warmup_btn.setEnabled(has_selection and not self.is_recording)
        self.record_toggle_btn.setEnabled(has_selection)
    
    def get_selected_devices(self):
        selected = []
        for device_idx, checkbox in self.device_checkboxes:
            if checkbox.isChecked():
                selected.append(device_idx)
        return selected
    
    def toggle_biast_for_device(self, device_idx, enabled):
        if enabled:
            self.log_message(f"Wlaczam BiasT dla urzadzenia [{device_idx}]...")
            self.log_message(f"Uruchamiam: rtl_biast -d {device_idx} -b 1")
            try:
                result = subprocess.run(['rtl_biast', '-d', str(device_idx), '-b', '1'], 
                                      capture_output=True, 
                                      text=True,
                                      timeout=5)
                if result.returncode == 0:
                    self.log_message(f"BiasT wlaczony dla urzadzenia [{device_idx}]")
                    if result.stdout.strip():
                        for line in result.stdout.strip().split('\n'):
                            self.log_message(f"{line}")
                else:
                    self.log_message(f"BLAD: Nie udalo sie wlaczyc BiasT dla [{device_idx}] (kod: {result.returncode})")
                    if result.stderr.strip():
                        self.log_message(f"{result.stderr.strip()}")
                    for idx, cb in self.device_biast_checkboxes:
                        if idx == device_idx:
                            cb.setChecked(False)
                            break
            except FileNotFoundError:
                self.log_message("BLAD: Nie znaleziono komendy rtl_biast")
                self.log_message("Zainstaluj: sudo apt install rtl-sdr")
                for idx, cb in self.device_biast_checkboxes:
                    if idx == device_idx:
                        cb.setChecked(False)
                        break
            except subprocess.TimeoutExpired:
                self.log_message(f"BLAD: Timeout podczas wlaczania BiasT dla [{device_idx}]")
                for idx, cb in self.device_biast_checkboxes:
                    if idx == device_idx:
                        cb.setChecked(False)
                        break
            except Exception as e:
                self.log_message(f"BLAD: {str(e)}")
                for idx, cb in self.device_biast_checkboxes:
                    if idx == device_idx:
                        cb.setChecked(False)
                        break
        else:
            self.log_message(f"Wylaczam BiasT dla urzadzenia [{device_idx}]...")
            self.log_message(f"Uruchamiam: rtl_biast -d {device_idx} -b 0")
            try:
                result = subprocess.run(['rtl_biast', '-d', str(device_idx), '-b', '0'], 
                                      capture_output=True, 
                                      text=True,
                                      timeout=5)
                if result.returncode == 0:
                    self.log_message(f"BiasT wylaczony dla urzadzenia [{device_idx}]")
                    if result.stdout.strip():
                        for line in result.stdout.strip().split('\n'):
                            self.log_message(f"{line}")
                else:
                    self.log_message(f"BLAD: Nie udalo sie wylaczyc BiasT dla [{device_idx}] (kod: {result.returncode})")
                    if result.stderr.strip():
                        self.log_message(f"{result.stderr.strip()}")
            except FileNotFoundError:
                self.log_message("BLAD: Nie znaleziono komendy rtl_biast")
                self.log_message("Zainstaluj: sudo apt install rtl-sdr")
            except subprocess.TimeoutExpired:
                self.log_message(f"BLAD: Timeout podczas wylaczania BiasT dla [{device_idx}]")
            except Exception as e:
                self.log_message(f"BLAD: {str(e)}")
    
    def get_spinbox_style(self):
        return """
        QDoubleSpinBox, QSpinBox {
            border: 2px solid #bdc3c7;
            border-radius: 5px;
            padding: 8px;
            font-size: 13px;
            background-color: white;
            color: #2c3e50;
            min-width: 120px;
        }
        QDoubleSpinBox:focus, QSpinBox:focus {
            border-color: #3498db;
        }
        QDoubleSpinBox:disabled, QSpinBox:disabled {
            background-color: #ecf0f1;
            color: #95a5a6;
            border-color: #bdc3c7;
        }
        """
    
    def log_message(self, message):
        self.log_signal.emit(message)
    
    def _log_message_safe(self, message):
        current_text = self.logs_text.toPlainText()
        self.logs_text.setPlainText(current_text + message + "\n")
        scrollbar = self.logs_text.verticalScrollBar()
        scrollbar.setValue(scrollbar.maximum())
    
    def warmup_receiver(self):
        selected_devices = self.get_selected_devices()
        
        if not selected_devices:
            self.log_message("Nie wybrano zadnego urzadzenia!")
            return
        
        self.warmup_btn.setEnabled(False)
        self.record_toggle_btn.setEnabled(False)
        self.scan_btn.setEnabled(False)
        self.frequency_spin.setEnabled(False)
        self.sample_rate_spin.setEnabled(False)
        self.filename_edit.setEnabled(False)
        
        for _, checkbox in self.device_checkboxes:
            checkbox.setEnabled(False)
        
        self.log_message(f"Rozpoczynam nagrzewanie {len(selected_devices)} urzadzenia/en...")
        
        def warmup_thread():
            processes = []
            device_errors = {}
            
            for device_idx in selected_devices:
                self.log_signal.emit(f"Urzadzenie [{device_idx}]: rtl_test -d {device_idx} -p")
                device_errors[device_idx] = {'found': False}
                
                try:
                    process = subprocess.Popen(['rtl_test', '-d', str(device_idx), '-p'], 
                                             stdout=subprocess.PIPE, 
                                             stderr=subprocess.STDOUT,
                                             text=True,
                                             bufsize=1, 
                                             universal_newlines=True)
                    processes.append((device_idx, process))
                    self.log_signal.emit(f"Urzadzenie [{device_idx}]: Proces uruchomiony (PID: {process.pid})")
                    
                    def read_output(proc, dev_idx):
                        try:
                            for line in proc.stdout:
                                line = line.strip()
                                if line:
                                    self.log_signal.emit(f"[{dev_idx}] {line}")
                                    if "No supported devices found" in line or "usb_claim_interface error" in line:
                                        device_errors[dev_idx]['found'] = True
                                        self.log_signal.emit(f"Urzadzenie [{dev_idx}]: Blad urzadzenia!")
                        except:
                            pass
                    
                    reader_thread = threading.Thread(target=read_output, args=(process, device_idx))
                    reader_thread.daemon = True
                    reader_thread.start()
                    
                except FileNotFoundError:
                    self.log_signal.emit(f"Urzadzenie [{device_idx}]: Nie znaleziono rtl_test")
                except Exception as e:
                    self.log_signal.emit(f"Urzadzenie [{device_idx}]: {str(e)}")
            
            for i in range(60, 0, -1):
                all_errors = all(device_errors[idx]['found'] for idx in selected_devices)
                if all_errors:
                    self.log_signal.emit("Przerwano nagrzewanie - bledy na wszystkich urzadzeniach")
                    break
                    
                if i % 10 == 0:
                    self.log_signal.emit(f"Pozostalo {i} sekund...")
                time.sleep(1)
            
            self.log_signal.emit("Zatrzymuje nagrzewanie...")
            for device_idx, process in processes:
                try:
                    self.log_signal.emit(f"Zatrzymuje urzadzenie [{device_idx}]...")
                    process.terminate()
                    
                    try:
                        process.wait(timeout=3)
                        self.log_signal.emit(f"Urzadzenie [{device_idx}]: Zakonczono")
                    except subprocess.TimeoutExpired:
                        self.log_signal.emit(f"Urzadzenie [{device_idx}]: Wymuszam zakonczenie...")
                        process.kill()
                        process.wait()
                except Exception as e:
                    self.log_signal.emit(f"Urzadzenie [{device_idx}]: {str(e)}")
            
            self.log_signal.emit("Nagrzewanie zakonczone!")
            self.warmup_complete.emit()
        
        thread = threading.Thread(target=warmup_thread)
        thread.daemon = True
        thread.start()
    
    def _warmup_finished(self):
        self.warmup_btn.setEnabled(True)
        self.scan_btn.setEnabled(True)
        self.frequency_spin.setEnabled(True)
        self.sample_rate_spin.setEnabled(True)
        self.filename_edit.setEnabled(True)
        
        # Włącz checkboxy urządzeń
        for _, checkbox in self.device_checkboxes:
            checkbox.setEnabled(True)
        
        self._update_button_states()
    
    def toggle_recording(self):
        if not self.is_recording:
            self.start_recording()
        else:
            self.stop_recording()
    
    def _update_timer_display(self, elapsed_seconds):
        minutes = elapsed_seconds // 60
        seconds = elapsed_seconds % 60
        self.timer_label.setText(f"Czas nagrywania: {minutes:02d}:{seconds:02d}")
    
    def start_recording(self):
        """Nagrywa tylko z wybranych urządzeń"""
        selected_devices = self.get_selected_devices()
        
        if not selected_devices:
            self.log_message("Nie wybrano zadnego urzadzenia!")
            return
        
        frequency_hz = int(self.frequency_spin.value() * 1000000)
        sample_rate_hz = int(self.sample_rate_spin.value() * 1000000)
        filename = self.filename_edit.text()
        
        self.log_message(f"Rozpoczynam nagrywanie z {len(selected_devices)} urzadzenia/en...")
        self.log_message(f"Czestotliwosc: {frequency_hz} Hz")
        self.log_message(f"Czestotliwosc probkowania: {sample_rate_hz} Hz")
        self.log_message(f"Wybrane urzadzenia: {selected_devices}")
        self.log_message(f"Nazwa pliku: {filename}")
        
        self.frequency_spin.setEnabled(False)
        self.sample_rate_spin.setEnabled(False)
        self.filename_edit.setEnabled(False)
        self.scan_btn.setEnabled(False)
        self.warmup_btn.setEnabled(False)
        
        # Wyłącz checkboxy podczas nagrywania
        for _, checkbox in self.device_checkboxes:
            checkbox.setEnabled(False)
        for _, checkbox in self.device_biast_checkboxes:
            checkbox.setEnabled(False)
        
        self.record_toggle_btn.setText("Wylacz nagrywanie")
        self.record_toggle_btn.setStyleSheet("""
        QPushButton {
            background-color: #e74c3c;
            color: white;
            border: none;
            border-radius: 5px;
            padding: 10px 15px;
            font-size: 12px;
            font-weight: bold;
        }
        QPushButton:hover {
            background-color: #c0392b;
        }
        QPushButton:pressed {
            background-color: #a93226;
        }
        """)
        
        self.is_recording = True
        self.recording_start_time = time.time()
        self.recording_processes = []
        
        for device_idx in selected_devices:
            base_name = filename.rsplit('.', 1)[0] if '.' in filename else filename
            output_file = f"{device_idx}_{base_name}.bin"
            
            # STAŁY GAIN: 38.2 dB (wyłącza AGC, zapobiega fałszywym alarmom z wielodrogowości)
            cmd = ['rtl_sdr', '-f', str(frequency_hz), '-s', str(sample_rate_hz), 
                   '-g', '38.2', '-d', str(device_idx), output_file]
            self.log_message(f"Urzadzenie [{device_idx}]: {' '.join(cmd)} [GAIN: 38.2 dB]")
            
            try:
                process = subprocess.Popen(cmd, 
                                         stdout=subprocess.PIPE, 
                                         stderr=subprocess.STDOUT,
                                         text=True,
                                         bufsize=1,
                                         universal_newlines=True)
                self.recording_processes.append((device_idx, process))
                self.log_message(f"Urzadzenie [{device_idx}] uruchomione (PID: {process.pid})")
                
                def read_sdr_output(proc, dev_idx):
                    try:
                        for line in proc.stdout:
                            line = line.strip()
                            if line:
                                self.log_signal.emit(f"[{dev_idx}] {line}")
                    except:
                        pass
                
                reader_thread = threading.Thread(target=read_sdr_output, args=(process, device_idx))
                reader_thread.daemon = True
                reader_thread.start()
                
            except FileNotFoundError:
                self.log_message(f"BLAD: Nie znaleziono komendy rtl_sdr")
                self.log_message("Zainstaluj: sudo apt install rtl-sdr")
                self.stop_recording()
                return
            except Exception as e:
                self.log_message(f"BLAD podczas uruchamiania urzadzenia [{device_idx}]: {str(e)}")

        def timer_thread():
            while self.is_recording:
                elapsed = int(time.time() - self.recording_start_time)
                self.update_timer.emit(elapsed)
                time.sleep(1)
        
        timer_t = threading.Thread(target=timer_thread)
        timer_t.daemon = True
        timer_t.start()
        
        self.log_message("Nagrywanie aktywne!")
    
    def stop_recording(self):
        self.log_message("Zatrzymuje nagrywanie...")
        
        self.is_recording = False
        
        for device_idx, process in self.recording_processes:
            try:
                self.log_message(f"Zatrzymuje urzadzenie [{device_idx}]...")
                process.terminate()
                
                try:
                    process.wait(timeout=3)
                    self.log_message(f"Urzadzenie [{device_idx}] zakonczone")
                except subprocess.TimeoutExpired:
                    self.log_message(f"Wymuszam zakonczenie urzadzenia [{device_idx}]...")
                    process.kill()
                    process.wait()
            except Exception as e:
                self.log_message(f"Blad podczas zatrzymywania urzadzenia [{device_idx}]: {str(e)}")
        
        self.recording_processes = []
        self.frequency_spin.setEnabled(True)
        self.sample_rate_spin.setEnabled(True)
        self.filename_edit.setEnabled(True)
        self.scan_btn.setEnabled(True)
        self.warmup_btn.setEnabled(True)

        # Włącz checkboxy
        for _, checkbox in self.device_checkboxes:
            checkbox.setEnabled(True)
        for _, checkbox in self.device_biast_checkboxes:
            checkbox.setEnabled(True)

        self.record_toggle_btn.setText("Wlacz nagrywanie")
        self.record_toggle_btn.setStyleSheet("""
        QPushButton {
            background-color: #27ae60;
            color: white;
            border: none;
            border-radius: 5px;
            padding: 10px 15px;
            font-size: 12px;
            font-weight: bold;
        }
        QPushButton:hover {
            background-color: #2ecc71;
        }
        QPushButton:pressed {
            background-color: #229954;
        }
        QPushButton:disabled {
            background-color: #95a5a6;
            color: #bdc3c7;
        }
        """)
        self.timer_label.setText("Czas nagrywania: 00:00")
        
        self.log_message("Nagrywanie zatrzymane!")
    
    def close_dialog(self):
        if self.is_recording:
            self.stop_recording()
        self.accept()
