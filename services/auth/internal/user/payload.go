package user

type ResponseUser struct {
	CreatedAt string
	UpdatedAt string
	Name      string
	Email     string
	UserUUID  string
}
type RequestRemoveUser struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=8,max=24"`
}
type RequestConfirm struct {
	Code       int    `validate:"required,min=100000,max=999999"`
	RefreshJwt string `json:"refresh_jwt"`
}

type RequestUpdateUser struct {
	NewName     string `json:"new_name" validate:"omitempty,min=2,max=64"`
	NewEmail    string `json:"new_email" validate:"omitempty,email"`
	NewPassword string `json:"new_password"  validate:"omitempty,min=8,max=24"`
	Password    string `validate:"omitempty,min=8,max=24,required_with=NewEmail NewPassword"`
	Email       string `validate:"omitempty,email"`
}
