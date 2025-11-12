from PySide6.QtWidgets import (QDialog, QVBoxLayout, QHBoxLayout, 
                             QLabel, QDoubleSpinBox, QSpinBox, 
                             QPushButton, QGroupBox, QGridLayout)
from PySide6.QtCore import Qt

class SettingsDialog(QDialog):
    def __init__(self, parent=None):
        super().__init__(parent)
        self.setWindowTitle("Ustawienia Analizy GPS")
        self.setModal(True)
        self.resize(400, 350)
        
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

        antenna_group = QGroupBox("Pozycje anten względem Anteny 1 [metry]")
        antenna_layout = QGridLayout(antenna_group)
        
        antenna_layout.addWidget(QLabel("<b>Antena</b>"), 0, 0)
        antenna_layout.addWidget(QLabel("<b>X [m]</b>"), 0, 1)
        antenna_layout.addWidget(QLabel("<b>Y [m]</b>"), 0, 2)

        antenna_layout.addWidget(QLabel("Antena 1 (ref):"), 1, 0)
        antenna_layout.addWidget(QLabel("0.0"), 1, 1)
        antenna_layout.addWidget(QLabel("0.0"), 1, 2)
        antenna_layout.addWidget(QLabel("Antena 2:"), 2, 0)
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
        
        antenna_layout.addWidget(QLabel("Antena 3:"), 3, 0)
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
        
        layout.addWidget(antenna_group)

        analysis_group = QGroupBox("Parametry Analizy")
        analysis_layout = QGridLayout(analysis_group)

        analysis_layout.addWidget(QLabel("Częstotliwość [MHz]:"), 0, 0)
        self.frequency = QDoubleSpinBox()
        self.frequency.setRange(1500, 1600)
        self.frequency.setValue(1575.42)  
        self.frequency.setDecimals(2)
        self.frequency.setSuffix(" MHz")
        self.frequency.setStyleSheet(self.get_spinbox_style())
        analysis_layout.addWidget(self.frequency, 0, 1)

        analysis_layout.addWidget(QLabel("Próg wykrywania [%]:"), 1, 0)
        self.threshold = QSpinBox()
        self.threshold.setRange(1, 100)
        self.threshold.setValue(30)
        self.threshold.setSuffix(" %")
        self.threshold.setStyleSheet(self.get_spinbox_style())
        analysis_layout.addWidget(self.threshold, 1, 1)

        analysis_layout.addWidget(QLabel("Częst. próbkowania [MHz]:"), 2, 0)
        self.sample_rate = QDoubleSpinBox()
        self.sample_rate.setRange(1, 50)
        self.sample_rate.setValue(2.048)
        self.sample_rate.setDecimals(3)
        self.sample_rate.setSuffix(" MHz")
        self.sample_rate.setStyleSheet(self.get_spinbox_style())
        analysis_layout.addWidget(self.sample_rate, 2, 1)
        
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
    
    def get_spinbox_style(self):
        """Zwraca styl dla pól SpinBox."""
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
        """
    
    def get_settings(self):
        """Zwraca słownik z ustawieniami."""
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
        
        return {
            'antenna_positions': antenna_positions,
            'antenna_distances': {
                '1_to_2': distance_12,
                '1_to_3': distance_13,
                '2_to_3': distance_23
            },
            'analysis_params': {
                'frequency': self.frequency.value(),
                'threshold': self.threshold.value(),
                'sample_rate': self.sample_rate.value()
            }
        }
    
    def set_settings(self, settings):
        """Ustawia wartości pól na podstawie słownika ustawień."""
        if 'antenna_positions' in settings:
            positions = settings['antenna_positions']
            # Antena 1 - punkt odniesienia
            antenna2_pos = positions.get('antenna2', [0.5, 0.0])
            antenna3_pos = positions.get('antenna3', [0.0, 0.5])
            
            self.antenna2_x.setValue(antenna2_pos[0])
            self.antenna2_y.setValue(antenna2_pos[1])
            self.antenna3_x.setValue(antenna3_pos[0])
            self.antenna3_y.setValue(antenna3_pos[1])
        
        if 'analysis_params' in settings:
            params = settings['analysis_params']
            self.frequency.setValue(params.get('frequency', 1575.42))
            self.threshold.setValue(params.get('threshold', 30))
            self.sample_rate.setValue(params.get('sample_rate', 2.048))