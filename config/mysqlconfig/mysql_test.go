package mysqlconfig

import (
	"testing"
)

func TestConfig_String(t *testing.T) {
	type fields struct {
		Host     string
		Port     string
		DB       string
		Username string
		Password string
		Charset  string
		Timeout  int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "test1",
			fields: fields{
				Username: "root",
				Password: "1234",
				DB:       "dbname",
			},
			want: "root:1234@(127.0.0.1:3306)/dbname?charset=utf8mb4&loc=Local&parseTime=True",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{
				Host:     tt.fields.Host,
				Port:     tt.fields.Port,
				DB:       tt.fields.DB,
				Username: tt.fields.Username,
				Password: tt.fields.Password,
				Charset:  tt.fields.Charset,
				Timeout:  tt.fields.Timeout,
			}
			if got := c.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
