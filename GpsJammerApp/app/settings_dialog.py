from PySide6.QtWidgets import (QDialog, QVBoxLayout, QHBoxLayout, 
                             QLabel, QDoubleSpinBox, QSpinBox, 
                             QPushButton, QGroupBox, QGridLayout, QMessageBox, QCheckBox)
from PySide6.QtCore import Qt, Signal
import os

class SettingsDialog(QDialog):
    position_changed = Signal(float, float, float)
    
    def __init__(self, parent=None, num_files=0, file_paths=None, config=None):
        super().__init__(parent)
        self.setWindowTitle("Ustawienia Analizy GPS — GPS Jammer Detection")
        self.setModal(True)
        self.resize(550, 500)
        self.num_files = num_files
        self.file_paths = file_paths if file_paths else []
        self.config = config
        
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
        layout.setContentsMargins(20, 20, 20, 20)
        layout.setSpacing(15)

        base_position_group = QGroupBox("Pozycja Anteny 1 (szerokość/długość geograficzna)")
        base_position_layout = QGridLayout(base_position_group)
        base_position_layout.setVerticalSpacing(10)
        base_position_layout.setHorizontalSpacing(10)
        
        base_position_layout.addWidget(QLabel("Szerokość [°]"), 0, 0)
        self.latitude_spin = QDoubleSpinBox()
        self.latitude_spin.setRange(-90.0, 90.0)
        self.latitude_spin.setValue(50.061430)
        self.latitude_spin.setDecimals(6)
        self.latitude_spin.setSuffix(" °")
        self.latitude_spin.setStyleSheet(self.get_spinbox_style())
        self.latitude_spin.valueChanged.connect(self.on_position_changed)
        base_position_layout.addWidget(self.latitude_spin, 0, 1)
        
        base_position_layout.addWidget(QLabel("Długość [°]"), 1, 0)
        self.longitude_spin = QDoubleSpinBox()
        self.longitude_spin.setRange(-180.0, 180.0)
        self.longitude_spin.setValue(19.936580)
        self.longitude_spin.setDecimals(6)
        self.longitude_spin.setSuffix(" °")
        self.longitude_spin.setStyleSheet(self.get_spinbox_style())
        self.longitude_spin.valueChanged.connect(self.on_position_changed)
        base_position_layout.addWidget(self.longitude_spin, 1, 1)
        
        base_position_layout.addWidget(QLabel("Wysokość [m]"), 2, 0)
        self.altitude_spin = QDoubleSpinBox()
        self.altitude_spin.setRange(-500.0, 10000.0)
        self.altitude_spin.setValue(0.00)
        self.altitude_spin.setDecimals(2)
        self.altitude_spin.setSuffix(" m")
        self.altitude_spin.setStyleSheet(self.get_spinbox_style())
        self.altitude_spin.valueChanged.connect(self.on_position_changed)
        base_position_layout.addWidget(self.altitude_spin, 2, 1)
        
        layout.addWidget(base_position_group)

        antenna_group = QGroupBox("Pozycje anten względem Anteny 1 [metry]")
        antenna_layout = QGridLayout(antenna_group)
        
        antenna_layout.addWidget(QLabel("<b>Antena</b>"), 0, 0)
        antenna_layout.addWidget(QLabel("<b>X [m]</b>"), 0, 1)
        antenna_layout.addWidget(QLabel("<b>Y [m]</b>"), 0, 2)

        self.antenna1_label = QLabel("Antena 1 (ref):")
        antenna_layout.addWidget(self.antenna1_label, 1, 0)
        self.antenna1_x_label = QLabel("0.0")
        antenna_layout.addWidget(self.antenna1_x_label, 1, 1)
        self.antenna1_y_label = QLabel("0.0")
        antenna_layout.addWidget(self.antenna1_y_label, 1, 2)
        self.antenna2_label = QLabel("Antena 2:")
        antenna_layout.addWidget(self.antenna2_label, 2, 0)
        self.antenna2_x = QDoubleSpinBox()
        self.antenna2_x.setRange(-50.0, 50.0)
        self.antenna2_x.setValue(0.5)
        self.antenna2_x.setDecimals(3)
        self.antenna2_x.setSuffix(" m")
        self.antenna2_x.setStyleSheet(self.get_spinbox_style())
        antenna_layout.addWidget(self.antenna2_x, 2, 1)
        self.antenna2_y = QDoubleSpinBox()
        self.antenna2_y.setRange(-50.0, 50.0)
        self.antenna2_y.setValue(0.0)
        self.antenna2_y.setDecimals(3)
        self.antenna2_y.setSuffix(" m")
        self.antenna2_y.setStyleSheet(self.get_spinbox_style())
        antenna_layout.addWidget(self.antenna2_y, 2, 2)
        
        self.antenna3_label = QLabel("Antena 3:")
        antenna_layout.addWidget(self.antenna3_label, 3, 0)
        self.antenna3_x = QDoubleSpinBox()
        self.antenna3_x.setRange(-50.0, 50.0)
        self.antenna3_x.setValue(0.0)  
        self.antenna3_x.setDecimals(3)
        self.antenna3_x.setSuffix(" m")
        self.antenna3_x.setStyleSheet(self.get_spinbox_style())
        antenna_layout.addWidget(self.antenna3_x, 3, 1)
        
        self.antenna3_y = QDoubleSpinBox()
        self.antenna3_y.setRange(-50.0, 50.0)
        self.antenna3_y.setValue(0.5)
        self.antenna3_y.setDecimals(3)
        self.antenna3_y.setSuffix(" m")
        self.antenna3_y.setStyleSheet(self.get_spinbox_style())
        antenna_layout.addWidget(self.antenna3_y, 3, 2)
        
        self.update_antenna_state()
        
        layout.addWidget(antenna_group)

        analysis_group = QGroupBox("Parametry Analizy")
        analysis_layout = QGridLayout(analysis_group)
        analysis_layout.setVerticalSpacing(15)

        analysis_layout.addWidget(QLabel("Częstotliwość:"), 0, 0)
        self.frequency_label = QLabel("1575.42 MHz")
        analysis_layout.addWidget(self.frequency_label, 0, 1)

        analysis_layout.addWidget(QLabel("Częstotliwość próbkowania:"), 1, 0)
        self.sample_rate_label = QLabel("2.048 MHz")
        analysis_layout.addWidget(self.sample_rate_label, 1, 1)

        analysis_layout.addWidget(QLabel("Utrzymuj pozycję:"), 2, 0)
        self.hold_position_checkbox = QCheckBox()
        self.hold_position_checkbox.setChecked(False)
        self.hold_position_checkbox.setStyleSheet("""
        QCheckBox {
            font-size: 13px;
            color: #2c3e50;
        }
        QCheckBox::indicator {
            width: 20px;
            height: 20px;
            border: 2px solid #bdc3c7;
            border-radius: 5px;
            background-color: white;
        }
        QCheckBox::indicator:checked {
            background-color: #3498db;
            border-color: #3498db;
            image: url(none);
        }
        QCheckBox::indicator:checked:after {
            content: "✓";
            color: white;
            font-weight: bold;
        }
        """)
        analysis_layout.addWidget(self.hold_position_checkbox, 2, 1)
        
        analysis_layout.addWidget(QLabel("Wykrywanie Spoofingu:"), 3, 0)
        self.spoofing_detection_checkbox = QCheckBox()
        self.spoofing_detection_checkbox.setChecked(False)
        self.spoofing_detection_checkbox.setStyleSheet("""
        QCheckBox {
            font-size: 13px;
            color: #2c3e50;
        }
        QCheckBox::indicator {
            width: 20px;
            height: 20px;
            border: 2px solid #bdc3c7;
            border-radius: 5px;
            background-color: white;
        }
        QCheckBox::indicator:checked {
            background-color: #e74c3c;
            border-color: #e74c3c;
            image: url(none);
        }
        QCheckBox::indicator:checked:after {
            content: "✓";
            color: white;
            font-weight: bold;
        }
        """)
        analysis_layout.addWidget(self.spoofing_detection_checkbox, 3, 1)
        
        layout.addWidget(analysis_group)
        
        button_layout = QHBoxLayout()
        
        self.ok_btn = QPushButton("Zapisz")
        self.ok_btn.clicked.connect(self.accept)
        self.ok_btn.setStyleSheet("""
        QPushButton {
            background-color: #27ae60;
            color: white;
            border: none;
            border-radius: 8px;
            padding: 12px 20px;
            font-size: 14px;
            font-weight: bold;
            margin: 5px;
        }
        QPushButton:hover {
            background-color: #2ecc71;
            box-shadow: 0 3px 6px rgba(0,0,0,0.2);
        }
        QPushButton:pressed {
            background-color: #229954;
        }
        """)
        
        self.cancel_btn = QPushButton("Anuluj")
        self.cancel_btn.clicked.connect(self.reject)
        self.cancel_btn.setStyleSheet("""
        QPushButton {
            background-color: #e74c3c;
            color: white;
            border: none;
            border-radius: 8px;
            padding: 12px 20px;
            font-size: 14px;
            font-weight: bold;
            margin: 5px;
        }
        QPushButton:hover {
            background-color: #c0392b;
            box-shadow: 0 3px 6px rgba(0,0,0,0.2);
        }
        QPushButton:pressed {
            background-color: #a93226;
        }
        """)
        
        button_layout.addWidget(self.cancel_btn)
        button_layout.addWidget(self.ok_btn)
        
        layout.addLayout(button_layout)
        
        if self.config:
            self.load_from_config()
    
    def on_position_changed(self):
        lat = self.latitude_spin.value()
        lon = self.longitude_spin.value()
        alt = self.altitude_spin.value()
        self.position_changed.emit(lat, lon, alt)
    
    def load_from_config(self):
        if hasattr(self.config, 'LAT'):
            self.latitude_spin.setValue(self.config.LAT)
        if hasattr(self.config, 'LNG'):
            self.longitude_spin.setValue(self.config.LNG)
        if hasattr(self.config, 'ALT'):
            self.altitude_spin.setValue(self.config.ALT)
    
    def update_antenna_state(self):
        disabled_label_style = "color: #95a5a6;"
        enabled_label_style = "color: #2c3e50; font-weight: bold;"
        
        if self.num_files == 0:
            self.antenna1_label.setStyleSheet(disabled_label_style)
            self.antenna1_x_label.setStyleSheet(disabled_label_style)
            self.antenna1_y_label.setStyleSheet(disabled_label_style)
            self.antenna2_label.setStyleSheet(disabled_label_style)
            self.antenna3_label.setStyleSheet(disabled_label_style)
            self.antenna2_x.setEnabled(False)
            self.antenna2_y.setEnabled(False)
            self.antenna3_x.setEnabled(False)
            self.antenna3_y.setEnabled(False)
        elif self.num_files == 1:
            self.antenna1_label.setStyleSheet(enabled_label_style)
            self.antenna1_x_label.setStyleSheet(enabled_label_style)
            self.antenna1_y_label.setStyleSheet(enabled_label_style)
            self.antenna2_label.setStyleSheet(disabled_label_style)
            self.antenna3_label.setStyleSheet(disabled_label_style)
            self.antenna2_x.setEnabled(False)
            self.antenna2_y.setEnabled(False)
            self.antenna3_x.setEnabled(False)
            self.antenna3_y.setEnabled(False)
        elif self.num_files == 2:
            self.antenna1_label.setStyleSheet(enabled_label_style)
            self.antenna1_x_label.setStyleSheet(enabled_label_style)
            self.antenna1_y_label.setStyleSheet(enabled_label_style)
            self.antenna2_label.setStyleSheet(enabled_label_style)
            self.antenna3_label.setStyleSheet(disabled_label_style)
            self.antenna2_x.setEnabled(True)
            self.antenna2_y.setEnabled(True)
            self.antenna3_x.setEnabled(False)
            self.antenna3_y.setEnabled(False)
        else:
            self.antenna1_label.setStyleSheet(enabled_label_style)
            self.antenna1_x_label.setStyleSheet(enabled_label_style)
            self.antenna1_y_label.setStyleSheet(enabled_label_style)
            self.antenna2_label.setStyleSheet(enabled_label_style)
            self.antenna3_label.setStyleSheet(enabled_label_style)
            self.antenna2_x.setEnabled(True)
            self.antenna2_y.setEnabled(True)
            self.antenna3_x.setEnabled(True)
            self.antenna3_y.setEnabled(True)
    
    def get_spinbox_style(self):
        return """
        QDoubleSpinBox, QSpinBox {
            border: 2px solid #bdc3c7;
            border-radius: 5px;
            padding: 8px;
            font-size: 13px;
            background-color: white;
            color: #2c3e50;
            min-width: 100px;
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
    
    def get_settings(self):
        import math

        antenna_positions = {
            'antenna1': [0.0, 0.0],
            'antenna2': [self.antenna2_x.value(), self.antenna2_y.value()],
            'antenna3': [self.antenna3_x.value(), self.antenna3_y.value()]
        }
        
        def calculate_distance(pos1, pos2):
            return math.sqrt((pos1[0] - pos2[0])**2 + (pos1[1] - pos2[1])**2)
        
        distance_12 = calculate_distance(antenna_positions['antenna1'], antenna_positions['antenna2'])
        distance_13 = calculate_distance(antenna_positions['antenna1'], antenna_positions['antenna3'])
        distance_23 = calculate_distance(antenna_positions['antenna2'], antenna_positions['antenna3'])
        
        frequency_text = self.frequency_label.text().replace(' MHz', '')
        sample_rate_text = self.sample_rate_label.text().replace(' MHz', '')
        
        return {
            'base_position': {
                'latitude': self.latitude_spin.value(),
                'longitude': self.longitude_spin.value(),
                'altitude': self.altitude_spin.value()
            },
            'antenna_positions': antenna_positions,
            'antenna_distances': {
                '1_to_2': distance_12,
                '1_to_3': distance_13,
                '2_to_3': distance_23
            },
            'analysis_params': {
                'frequency': float(frequency_text),
                'sample_rate': float(sample_rate_text),
                'hold_position': self.hold_position_checkbox.isChecked(),
                'spoofing_detection': self.spoofing_detection_checkbox.isChecked()
            }
        }
    
    def set_settings(self, settings):
        if 'base_position' in settings:
            base_pos = settings['base_position']
            self.latitude_spin.setValue(base_pos.get('latitude', 50.061430))
            self.longitude_spin.setValue(base_pos.get('longitude', 19.936580))
            self.altitude_spin.setValue(base_pos.get('altitude', 0.0))
        
        if 'antenna_positions' in settings:
            positions = settings['antenna_positions']
            antenna2_pos = positions.get('antenna2', [0.5, 0.0])
            antenna3_pos = positions.get('antenna3', [0.0, 0.5])
            
            self.antenna2_x.setValue(antenna2_pos[0])
            self.antenna2_y.setValue(antenna2_pos[1])
            self.antenna3_x.setValue(antenna3_pos[0])
            self.antenna3_y.setValue(antenna3_pos[1])
        
        if 'analysis_params' in settings:
            params = settings['analysis_params']
            
            hold_position = params.get('hold_position', False)
            self.hold_position_checkbox.setChecked(hold_position)
            
            spoofing_detection = params.get('spoofing_detection', False)
            self.spoofing_detection_checkbox.setChecked(spoofing_detection)
            
            frequency = params.get('frequency', 1575.42)
            sample_rate = params.get('sample_rate', 2.048)
            self.frequency_label.setText(f"{frequency:.2f} MHz")
            self.sample_rate_label.setText(f"{sample_rate:.3f} MHz")