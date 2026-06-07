# 测试特征依恋检测

class UserInfo:
    """数据类，存储用户信息"""
    def __init__(self, name, email, age, address, phone):
        self.name = name
        self.email = email
        self.age = age
        self.address = address
        self.phone = phone


class Order:
    """订单类"""
    def __init__(self, order_id, total, status):
        self.order_id = order_id
        self.total = total
        self.status = status
        self.items = []


class ReportGenerator:
    """报告生成器 - 这个类的方法会大量访问其他类的属性，触发特征依恋"""
    
    def __init__(self, config):
        self.config = config
        self.output_dir = "./reports"
    
    def generate_user_report(self, user, order, output_format):
        """这个方法会大量访问user和order的属性"""
        report = []
        
        # 大量访问user的属性
        report.append(f"用户姓名: {user.name}")
        report.append(f"用户邮箱: {user.email}")
        report.append(f"用户年龄: {user.age}")
        report.append(f"用户地址: {user.address}")
        report.append(f"用户电话: {user.phone}")
        
        # 大量访问order的属性
        report.append(f"订单ID: {order.order_id}")
        report.append(f"订单总额: {order.total}")
        report.append(f"订单状态: {order.status}")
        report.append(f"订单商品数: {len(order.items)}")
        
        # 更多访问
        if user.age >= 18:
            report.append(f"成年用户: {user.name}")
        if user.address:
            report.append(f"配送地址: {user.address}")
        if order.total > 100:
            report.append(f"大额订单: {order.total}")
        
        # 只访问了一次自身属性
        report.append(f"输出目录: {self.output_dir}")
        
        return "\n".join(report)
    
    def generate_summary(self, user_list, order_list):
        """同样大量访问其他类的属性"""
        summary = []
        
        total_amount = 0
        active_users = 0
        
        for user in user_list:
            if user.age >= 18:
                active_users += 1
            summary.append(f"{user.name} ({user.email})")
        
        for order in order_list:
            total_amount += order.total
            if order.status == "completed":
                summary.append(f"已完成: {order.order_id} - ${order.total}")
        
        summary.append(f"报告输出目录: {self.output_dir}")
        
        return summary
